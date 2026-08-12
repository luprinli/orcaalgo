package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	mw "github.com/lee-econ/orca-core/internal/api/middleware"
	"github.com/lee-econ/orca-core/internal/db"
	"github.com/lee-econ/orca-core/internal/email"
	"github.com/lee-econ/orca-core/internal/security"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	repo              *db.Repository
	emailService      email.EmailService
	frontendURL       string
	loginAttempts     map[string]int
	loginBlockedUntil map[string]time.Time
	loginMu           sync.Mutex
}

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	TOTPCode string `json:"totp_code"`
}

type registerReq struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=8"`
	Email    string `json:"email"`
}

type forgotPasswordReq struct {
	Email string `json:"email" binding:"required"`
}

type resetPasswordReq struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type totpSetupResp struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

func NewAuthHandlerWithRepo(repo *db.Repository, emailSvc email.EmailService, frontendURL string) *AuthHandler {
	h := &AuthHandler{
		repo:              repo,
		emailService:      emailSvc,
		frontendURL:       frontendURL,
		loginAttempts:     make(map[string]int),
		loginBlockedUntil: make(map[string]time.Time),
	}
	h.migrateAdminUser()
	go h.loginCleanupLoop()
	return h
}

func NewAuthHandler() *AuthHandler {
	h := &AuthHandler{
		loginAttempts:     make(map[string]int),
		loginBlockedUntil: make(map[string]time.Time),
	}
	h.ensureAdminUser()
	go h.loginCleanupLoop()
	return h
}

func NewAuthHandlerLegacy(repo *db.Repository) *AuthHandler {
	h := &AuthHandler{
		repo:              repo,
		loginAttempts:     make(map[string]int),
		loginBlockedUntil: make(map[string]time.Time),
	}
	h.migrateAdminUser()
	go h.loginCleanupLoop()
	return h
}

func (h *AuthHandler) ensureAdminUser() {
	adminPass := os.Getenv("ORCA_ADMIN_PASSWORD")
	if adminPass == "" {
		return
	}
	if h.repo != nil {
		h.migrateAdminUser()
		return
	}
}

func (h *AuthHandler) migrateAdminUser() {
	if h.repo == nil {
		return
	}

	adminPass := os.Getenv("ORCA_ADMIN_PASSWORD")
	if adminPass == "" {
		return
	}

	ctx := context.Background()

	count, err := h.repo.UserCount(ctx)
	if err != nil {
		slog.Error("failed to check user count", "error", err, "component", "auth")
		return
	}
	if count > 0 {
		return
	}

	adminHash, err := hashPassword(adminPass)
	if err != nil {
		slog.Error("failed to hash admin password", "error", err, "component", "auth")
		return
	}
	adminUser := &db.DBUser{
		ID:           uuid.NewString(),
		Username:     "admin",
		Email:        "admin@orca.local",
		PasswordHash: adminHash,
		Roles:        []string{"admin", "trader"},
		IsVerified:   true,
		IsActive:     true,
	}

	if err := h.repo.CreateUser(ctx, adminUser); err != nil {
		slog.Error("failed to migrate admin user", "error", err, "component", "auth")
		return
	}
	slog.Info("admin user migrated to database", "component", "auth")
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.isLoginBlocked(req.Username) {
		h.loginMu.Lock()
		until := h.loginBlockedUntil[req.Username]
		h.loginMu.Unlock()
		remain := time.Until(until).Minutes()
		c.JSON(http.StatusTooManyRequests, gin.H{"error": fmt.Sprintf("Too many login attempts. Try again in %.0f minutes.", remain)})
		return
	}

	if h.repo != nil {
		h.loginFromDB(c, req)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user store not available"})
	}
}

func (h *AuthHandler) loginFromDB(c *gin.Context, req loginReq) {
	ctx := c.Request.Context()
	user, err := h.repo.GetUserByUsername(ctx, req.Username)
	if err != nil || !checkPasswordHash(req.Password, user.PasswordHash) {
		h.recordFailedLogin(req.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is disabled"})
		return
	}

	if user.TOTPEnabled {
		if req.TOTPCode == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "2FA code required"})
			return
		}
		if valid, _ := security.ValidateTOTP(user.TOTPSecret, req.TOTPCode); !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid 2FA code"})
			return
		}
	}

	h.clearFailedLogins(req.Username)

	pair, err := security.GenerateTokenPair(user.ID, user.Username, user.Roles, jwtSecret(), 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": pair.AccessToken, "refresh_token": pair.RefreshToken,
		"expires_in": pair.ExpiresIn, "token_type": pair.TokenType,
		"username": user.Username, "roles": user.Roles,
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	emailAddr := req.Email
	if emailAddr == "" {
		emailAddr = req.Username + "@orca.local"
	}

	if h.repo != nil {
		h.registerInDB(c, req, emailAddr)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user store not available"})
	}
}

func (h *AuthHandler) registerInDB(c *gin.Context, req registerReq, emailAddr string) {
	ctx := c.Request.Context()

	existing, _ := h.repo.GetUserByUsername(ctx, req.Username)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username exists"})
		return
	}

	existingEmail, _ := h.repo.GetUserByEmail(ctx, emailAddr)
	if existingEmail != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	userHash, err := hashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}
	user := &db.DBUser{
		ID:           uuid.NewString(),
		Username:     req.Username,
		Email:        emailAddr,
		PasswordHash: userHash,
		Roles:        []string{"trader"},
		IsVerified:   false,
		IsActive:     true,
	}

	if err := h.repo.CreateUser(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	slog.Info("user registered", "username", user.Username, "component", "auth")
	c.JSON(http.StatusCreated, gin.H{"username": req.Username, "message": "Registration successful"})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user store not available"})
		return
	}

	ctx := c.Request.Context()
	user, err := h.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link has been sent"})
		return
	}

	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	tokenStr := hex.EncodeToString(tokenBytes)

	resetToken := &db.PasswordResetToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := h.repo.CreatePasswordResetToken(ctx, resetToken); err != nil {
		slog.Error("failed to create reset token", "error", err, "component", "auth")
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link has been sent"})
		return
	}

	resetLink := h.frontendURL + "/reset-password?token=" + tokenStr

	if h.emailService != nil {
		if err := h.emailService.SendTemplate([]string{req.Email}, "Orca Algo - Password Reset", "password_reset", map[string]interface{}{
			"ResetLink": resetLink,
			"ExpiresIn": "1 hour",
		}); err != nil {
			slog.Error("failed to send password reset email", "error", err, "component", "auth")
		}
	} else {
		slog.Info("password reset token created", "email", req.Email, "token", tokenStr, "component", "auth")
	}

	c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link has been sent"})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user store not available"})
		return
	}

	ctx := c.Request.Context()
	token, err := h.repo.GetPasswordResetToken(ctx, req.Token)
	if err != nil || token.Used || time.Now().After(token.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
		return
	}

	newHash, err := hashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}
	if err := h.repo.UpdateUserPassword(ctx, token.UserID, newHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	h.repo.MarkResetTokenUsed(ctx, token.ID)

	slog.Info("password reset completed", "user_id", token.UserID, "component", "auth")
	c.JSON(http.StatusOK, gin.H{"message": "Password has been reset successfully"})
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token required"})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user store not available"})
		return
	}

	ctx := c.Request.Context()
	token, err := h.repo.GetEmailVerificationToken(ctx, tokenStr)
	if err != nil || token.Used || time.Now().After(token.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired verification token"})
		return
	}

	if err := h.repo.MarkUserVerified(ctx, token.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify email"})
		return
	}

	h.repo.MarkVerificationTokenUsed(ctx, token.ID)

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

func (h *AuthHandler) ResendVerification(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authentication required"})
		return
	}

	if h.repo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user store not available"})
		return
	}

	ctx := c.Request.Context()
	user, err := h.repo.GetUserByUsername(ctx, username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.IsVerified {
		c.JSON(http.StatusOK, gin.H{"message": "Email already verified"})
		return
	}

	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	tokenStr := hex.EncodeToString(tokenBytes)

	verifyToken := &db.EmailVerificationToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		Token:     tokenStr,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := h.repo.CreateEmailVerificationToken(ctx, verifyToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create verification token"})
		return
	}

	verifyLink := h.frontendURL + "/verify-email?token=" + tokenStr

	if h.emailService != nil {
		if err := h.emailService.SendTemplate([]string{user.Email}, "Orca Algo - Verify Your Email", "email_verification", map[string]interface{}{
			"VerifyLink": verifyLink,
			"ExpiresIn":  "24 hours",
		}); err != nil {
			slog.Error("failed to send verification email", "error", err, "component", "auth")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send verification email"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification email sent"})
}

func (h *AuthHandler) SetupTOTP(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	if h.repo != nil {
		h.setupTOTPWithDB(c, username)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user store not available"})
	}
}

func (h *AuthHandler) setupTOTPWithDB(c *gin.Context, username string) {
	ctx := c.Request.Context()
	user, err := h.repo.GetUserByUsername(ctx, username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	secret := security.GenerateTOTPSecret("Orca Core", username)

	if err := h.repo.UpdateUserTOTP(ctx, user.ID, secret.Secret, true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save 2FA settings"})
		return
	}

	c.JSON(http.StatusOK, totpSetupResp{Secret: secret.Secret, URI: security.GetTOTPURI(secret)})
}

func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/register", h.Register)
		auth.POST("/forgot-password", h.ForgotPassword)
		auth.POST("/reset-password", h.ResetPassword)
		auth.GET("/verify-email", h.VerifyEmail)
		auth.POST("/resend-verification", mw.AuthMiddleware(), h.ResendVerification)
		auth.POST("/2fa/setup", mw.AuthMiddleware(), h.SetupTOTP)
	}
}

func (h *AuthHandler) ValidateTOTP(username, code string) bool {
	if h.repo == nil {
		return false
	}
	ctx := context.Background()
	user, err := h.repo.GetUserByUsername(ctx, username)
	if err != nil || !user.TOTPEnabled {
		return false
	}
	valid, _ := security.ValidateTOTP(user.TOTPSecret, code)
	return valid
}

func (h *AuthHandler) isLoginBlocked(username string) bool {
	h.loginMu.Lock()
	defer h.loginMu.Unlock()
	blockedUntil, ok := h.loginBlockedUntil[username]
	if !ok {
		return false
	}
	if time.Now().Before(blockedUntil) {
		return true
	}
	delete(h.loginBlockedUntil, username)
	delete(h.loginAttempts, username)
	return false
}

func (h *AuthHandler) recordFailedLogin(username string) {
	h.loginMu.Lock()
	defer h.loginMu.Unlock()
	h.loginAttempts[username]++
	if h.loginAttempts[username] >= 5 {
		h.loginBlockedUntil[username] = time.Now().Add(15 * time.Minute)
	}
}

func (h *AuthHandler) clearFailedLogins(username string) {
	h.loginMu.Lock()
	defer h.loginMu.Unlock()
	delete(h.loginAttempts, username)
	delete(h.loginBlockedUntil, username)
}

func (h *AuthHandler) loginCleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		h.loginMu.Lock()
		now := time.Now()
		for username, until := range h.loginBlockedUntil {
			if now.After(until) {
				delete(h.loginBlockedUntil, username)
				delete(h.loginAttempts, username)
			}
		}
		h.loginMu.Unlock()
	}
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt: failed to hash password: %w", err)
	}
	return string(hash), nil
}

func jwtSecret() []byte {
	return mw.GetJWTSecret()
}

func checkPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

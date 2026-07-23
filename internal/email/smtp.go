package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

type SMTPEmailService struct {
	config SMTPConfig
}

func NewSMTPEmailService(cfg SMTPConfig) *SMTPEmailService {
	if cfg.Host == "" {
		cfg.Host = os.Getenv("SMTP_HOST")
	}
	if cfg.Port == 0 {
		p, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
		if p > 0 {
			cfg.Port = p
		} else {
			cfg.Port = 587
		}
	}
	if cfg.Username == "" {
		cfg.Username = os.Getenv("SMTP_USER")
	}
	if cfg.Password == "" {
		cfg.Password = os.Getenv("SMTP_PASS")
	}
	if cfg.From == "" {
		cfg.From = os.Getenv("SMTP_FROM")
	}
	return &SMTPEmailService{config: cfg}
}

func NewSMTPFromEnv() *SMTPEmailService {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if port == 0 {
		port = 587
	}
	return NewSMTPEmailService(SMTPConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     port,
		Username: os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASS"),
		From:     os.Getenv("SMTP_FROM"),
		FromName: os.Getenv("SMTP_FROM_NAME"),
	})
}

func (s *SMTPEmailService) Send(to []string, subject, body string, isHTML bool) error {
	if s.config.Host == "" || s.config.From == "" {
		return fmt.Errorf("SMTP not configured: missing host or from address")
	}

	contentType := "text/plain"
	if isHTML {
		contentType = "text/html"
	}

	fromDisplay := s.config.FromName
	if fromDisplay == "" {
		fromDisplay = "Orca Algo"
	}

	msg := buildMessage(fromDisplay+" <"+s.config.From+">", to, subject, contentType, body)

	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	if s.config.Port == 465 {
		return s.sendSMTPOverTLS(addr, auth, s.config.From, to, msg)
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: s.config.Host, InsecureSkipVerify: false}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", rcpt, err)
		}
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	defer wc.Close()

	if _, err := wc.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}

	log.Printf("email sent to %v: %s", to, subject)
	return nil
}

func (s *SMTPEmailService) SendTemplate(to []string, subject, templateName string, data map[string]interface{}) error {
	tmplStr := defaultTemplates[templateName]
	if tmplStr == "" {
		return fmt.Errorf("unknown email template: %s", templateName)
	}

	tmpl, err := template.New(templateName).Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("parse template %s: %w", templateName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template %s: %w", templateName, err)
	}

	return s.Send(to, subject, buf.String(), true)
}

func (s *SMTPEmailService) TestConnection() error {
	if s.config.Host == "" {
		return fmt.Errorf("SMTP host not configured")
	}
	addr := net.JoinHostPort(s.config.Host, fmt.Sprintf("%d", s.config.Port))
	conn, err := net.DialTimeout("tcp", addr, smtpTimeout)
	if err != nil {
		return fmt.Errorf("cannot connect to SMTP server: %w", err)
	}
	conn.Close()
	return nil
}

func (s *SMTPEmailService) Config() SMTPConfig {
	return s.config
}

func (s *SMTPEmailService) IsConfigured() bool {
	return s.config.Host != "" && s.config.From != ""
}

func (s *SMTPEmailService) sendSMTPOverTLS(addr string, auth smtp.Auth, from string, to []string, msg string) error {
	tlsConfig := &tls.Config{ServerName: s.config.Host, InsecureSkipVerify: false}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt: %w", err)
		}
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	defer wc.Close()

	if _, err := wc.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}

	return nil
}

const smtpTimeout = 0

var defaultTemplates = map[string]string{
	"password_reset": `<!DOCTYPE html>
<html><body>
<h2>Password Reset</h2>
<p>You requested a password reset. Click the link below to set a new password:</p>
<p><a href="{{.ResetLink}}">{{.ResetLink}}</a></p>
<p>This link expires in {{.ExpiresIn}}. If you did not request this, ignore this email.</p>
</body></html>`,

	"email_verification": `<!DOCTYPE html>
<html><body>
<h2>Verify Your Email</h2>
<p>Welcome to Orca Algo! Click the link below to verify your email address:</p>
<p><a href="{{.VerifyLink}}">{{.VerifyLink}}</a></p>
<p>This link expires in {{.ExpiresIn}}.</p>
</body></html>`,

	"notification": `<!DOCTYPE html>
<html><body>
<h2>{{.Title}}</h2>
<p>{{.Message}}</p>
{{if .Details}}<pre>{{.Details}}</pre>{{end}}
<p><small>Sent by Orca Algo Notification System</small></p>
</body></html>`,
}

func buildMessage(from string, to []string, subject, contentType, body string) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString(fmt.Sprintf("MIME-Version: 1.0\r\n"))
	buf.WriteString(fmt.Sprintf("Content-Type: %s; charset=\"UTF-8\"\r\n", contentType))
	buf.WriteString("\r\n")
	buf.WriteString(body)
	return buf.String()
}

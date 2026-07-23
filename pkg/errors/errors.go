package errors

import "time"

type Category string

const (
	CategoryConfig   Category = "config"
	CategoryNetwork  Category = "network"
	CategoryBroker   Category = "broker"
	CategoryData     Category = "data"
	CategoryAuth     Category = "auth"
	CategoryInternal Category = "internal"
	CategoryExternal Category = "external"
)

type Severity string

const (
	SeverityDebug    Severity = "debug"
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

type AppError struct {
	Category   ErrorCategory  `json:"category"`
	Severity   ErrorSeverity  `json:"severity"`
	Message    string         `json:"message"`
	UserAction string         `json:"user_action,omitempty"`
	Err        error          `json:"-"`
	Timestamp  time.Time      `json:"timestamp"`
	UserID     string         `json:"user_id,omitempty"`
	Component  string         `json:"component"`
	Retryable  bool           `json:"retryable"`
}

// Type aliases for cleaner usage
type ErrorCategory = Category
type ErrorSeverity = Severity

func New(category Category, severity Severity, component, message string, err error) *AppError {
	return &AppError{
		Category:  category,
		Severity:  severity,
		Message:   message,
		Component: component,
		Err:       err,
		Timestamp: time.Now(),
	}
}

func NewWithUser(category Category, severity Severity, component, message string, err error, userID string) *AppError {
	e := New(category, severity, component, message, err)
	e.UserID = userID
	return e
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) WithUserAction(action string) *AppError {
	e.UserAction = action
	return e
}

func (e *AppError) WithRetry() *AppError {
	e.Retryable = true
	return e
}

func (e *AppError) IsCritical() bool {
	return e.Severity == SeverityCritical
}

func (e *AppError) IsError() bool {
	return e.Severity == SeverityError || e.Severity == SeverityCritical
}

func (e *AppError) ShouldNotify() bool {
	return e.Severity == SeverityWarning || e.Severity == SeverityError || e.Severity == SeverityCritical
}

func ConfigError(component, message string, err error) *AppError {
	return New(CategoryConfig, SeverityError, component, message, err).WithUserAction("Check configuration in Settings.")
}

func NetworkError(component, message string, err error) *AppError {
	return New(CategoryNetwork, SeverityError, component, message, err).WithRetry().WithUserAction("Check network connectivity and try again.")
}

func BrokerError(component, message string, err error) *AppError {
	return New(CategoryBroker, SeverityError, component, message, err).WithUserAction("Verify broker credentials in Accounts.")
}

func AuthError(component, message string, err error) *AppError {
	return New(CategoryAuth, SeverityError, component, message, err)
}

func InternalError(component, message string, err error) *AppError {
	return New(CategoryInternal, SeverityError, component, message, err).WithUserAction("Contact support if this persists.")
}

func Warning(component, message string, err error) *AppError {
	return New(CategoryInternal, SeverityWarning, component, message, err)
}

func Critical(component, message string, err error) *AppError {
	return New(CategoryInternal, SeverityCritical, component, message, err)
}

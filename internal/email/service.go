package email

type EmailService interface {
	Send(to []string, subject, body string, isHTML bool) error
	SendTemplate(to []string, subject, templateName string, data map[string]interface{}) error
}

type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	FromName string `json:"from_name"`
}

type SendResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

package notify

import (
	"log"

	"github.com/lee-econ/orca-core/internal/email"
)

type EmailNotifier struct {
	service  email.EmailService
	toEmails []string
	enabled  bool
}

func NewEmailNotifier(svc email.EmailService, toEmails ...string) *EmailNotifier {
	return &EmailNotifier{
		service:  svc,
		toEmails: toEmails,
		enabled:  svc != nil && len(toEmails) > 0,
	}
}

func (e *EmailNotifier) Name() string {
	return "email"
}

func (e *EmailNotifier) IsEnabled() bool {
	return e.enabled
}

func (e *EmailNotifier) Send(event Event) error {
	if !e.enabled {
		return nil
	}

	subject := "Orca Algo: " + string(event.Type)
	if event.Level == LevelCritical {
		subject = "[CRITICAL] " + subject
	} else if event.Level == LevelWarning {
		subject = "[WARNING] " + subject
	}

	err := e.service.SendTemplate(e.toEmails, subject, "notification", map[string]interface{}{
		"Title":   event.Title,
		"Message": event.Message,
		"Details": event.Details,
	})
	if err != nil {
		log.Printf("email notifier: failed to send: %v", err)
		return err
	}

	return nil
}

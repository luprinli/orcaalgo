package email

import (
	"testing"
)

func TestMockEmailServiceSend(t *testing.T) {
	svc := NewMockEmailService()

	err := svc.Send([]string{"test@example.com"}, "Test Subject", "Test Body", false)
	if err != nil {
		t.Fatalf("Send should succeed: %v", err)
	}
	if svc.SentCount() != 1 {
		t.Errorf("expected 1 sent email, got %d", svc.SentCount())
	}

	last := svc.LastSent()
	if last == nil {
		t.Fatal("expected last sent email")
	}
	if last.Subject != "Test Subject" {
		t.Errorf("expected subject 'Test Subject', got %s", last.Subject)
	}
	if last.To[0] != "test@example.com" {
		t.Errorf("expected to 'test@example.com', got %s", last.To[0])
	}
}

func TestMockEmailServiceFail(t *testing.T) {
	svc := NewMockEmailService()
	svc.FailNext = true
	svc.FailErr = "simulated failure"

	err := svc.Send([]string{"test@example.com"}, "Test", "Body", false)
	if err == nil {
		t.Error("expected error on FailNext")
	}
	if svc.SentCount() != 0 {
		t.Errorf("expected 0 sent emails on failure, got %d", svc.SentCount())
	}
}

func TestMockEmailServiceSendTemplate(t *testing.T) {
	svc := NewMockEmailService()

	err := svc.SendTemplate([]string{"user@example.com"}, "Welcome", "password_reset", map[string]interface{}{
		"ResetLink": "https://example.com/reset?token=abc",
	})
	if err != nil {
		t.Fatalf("SendTemplate should succeed: %v", err)
	}

	last := svc.LastSent()
	if last == nil {
		t.Fatal("expected last sent email")
	}
	if !last.IsHTML {
		t.Error("expected HTML email")
	}
}

func TestMockEmailServiceClearSent(t *testing.T) {
	svc := NewMockEmailService()
	svc.Send([]string{"a@b.com"}, "S1", "B1", false)
	svc.Send([]string{"a@b.com"}, "S2", "B2", false)

	if svc.SentCount() != 2 {
		t.Errorf("expected 2 sent emails, got %d", svc.SentCount())
	}

	svc.ClearSent()
	if svc.SentCount() != 0 {
		t.Errorf("expected 0 after clear, got %d", svc.SentCount())
	}
}

func TestSMTPEmailServiceBuildMessage(t *testing.T) {
	msg := buildMessage("Orca <orca@test.com>", []string{"user@test.com"}, "Subject Line", "text/html", "<p>hello</p>")
	if msg == "" {
		t.Error("expected non-empty message")
	}
}

func TestSMTPConfigFromEnv(t *testing.T) {
	svc := NewSMTPFromEnv()
	if svc == nil {
		t.Error("expected non-nil service even without env vars")
	}

	if svc.IsConfigured() {
		t.Log("SMTP configured from env, tests may send real mail")
	} else {
		t.Log("SMTP not configured from env, skipping connection test")
	}
}

func TestDefaultTemplates(t *testing.T) {
	if defaultTemplates["password_reset"] == "" {
		t.Error("password_reset template should not be empty")
	}
	if defaultTemplates["email_verification"] == "" {
		t.Error("email_verification template should not be empty")
	}
	if defaultTemplates["notification"] == "" {
		t.Error("notification template should not be empty")
	}
}

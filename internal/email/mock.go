package email

import (
	"sync"
	"time"
)

type MockEmailService struct {
	mu       sync.Mutex
	Sent     []MockEmail
	FailNext bool
	FailErr  string
}

type MockEmail struct {
	To       []string
	Subject  string
	Body     string
	IsHTML   bool
	SentAt   time.Time
}

func NewMockEmailService() *MockEmailService {
	return &MockEmailService{}
}

func (m *MockEmailService) Send(to []string, subject, body string, isHTML bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.FailNext {
		m.FailNext = false
		return &sendError{m.FailErr}
	}

	m.Sent = append(m.Sent, MockEmail{
		To:      to,
		Subject: subject,
		Body:    body,
		IsHTML:  isHTML,
		SentAt:  time.Now(),
	})
	return nil
}

func (m *MockEmailService) SendTemplate(to []string, subject, templateName string, data map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.FailNext {
		m.FailNext = false
		return &sendError{m.FailErr}
	}

	m.Sent = append(m.Sent, MockEmail{
		To:      to,
		Subject: subject,
		Body:    templateName,
		IsHTML:  true,
		SentAt:  time.Now(),
	})
	return nil
}

func (m *MockEmailService) ClearSent() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sent = nil
}

func (m *MockEmailService) SentCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Sent)
}

func (m *MockEmailService) LastSent() *MockEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Sent) == 0 {
		return nil
	}
	return &m.Sent[len(m.Sent)-1]
}

type sendError struct {
	msg string
}

func (e *sendError) Error() string {
	return e.msg
}

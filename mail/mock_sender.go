package mail

import "github.com/rs/zerolog/log"

// MockSender mocks email send functionality by saving the email contents
// in the specified directory and skips the file attachments
type MockSender struct {
}

func NewMockSender(path string) EmailSender {
	return &MockSender{}
}

func (m *MockSender) SendEmail(subject string, content string, to []string, cc []string, bcc []string, attachFiles []string) error {
	log.Info().
		Str("subject", subject).
		Str("content", content).
		Str("to", to[0]).
		Msg("Verification email sent")

	return nil
}

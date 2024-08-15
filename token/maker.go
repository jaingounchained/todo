package token

import "time"

// Maker is an interface for managing tokens
type Maker interface {
	// CreateToken created a new token for a specific username and durations
	CreateToken(username string, duration time.Duration) (string, *Payload, error)

	// VerifyToken checks if the token is valid
	VerifyToken(token string) (*Payload, error)
}

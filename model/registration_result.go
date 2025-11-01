package model

type RegistrationResult struct {
	UserID   uint64
	Username string
	RawPass  string

	Session *Session
}

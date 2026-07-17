package store

import "errors"

var (
	ErrNotFound               = errors.New("not found")
	ErrEnrollmentPending      = errors.New("enrollment is waiting for approval")
	ErrEnrollmentExpired      = errors.New("enrollment has expired")
	ErrEnrollmentRejected     = errors.New("enrollment was rejected")
	ErrEnrollmentRevoked      = errors.New("enrollment was revoked")
	ErrEnrollmentClaimed      = errors.New("enrollment was already claimed")
	ErrEnrollmentConflict     = errors.New("enrollment conflicts with an existing agent")
	ErrInvalidAgentCredential = errors.New("invalid agent credential")
)

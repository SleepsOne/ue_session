package domain

import (
	"context"
	"time"
)

// Session represents a UE session in the 5G Core network
type Session struct {
	TMSI         string          `json:"tmsi" redis:"tmsi"`
	IMSI         string          `json:"imsi" redis:"imsi"`
	MSISDN       string          `json:"msisdn" redis:"msisdn"`
	AttachTime   time.Time       `json:"attach_time" redis:"attach_time"`
	LastUpdate   time.Time       `json:"last_update" redis:"last_update"`
	GNBID        string          `json:"gnb_id" redis:"gnb_id"`
	TAI          string          `json:"tai" redis:"tai"`
	UEState      string          `json:"ue_state" redis:"ue_state"`
	Capabilities []string        `json:"capabilities" redis:"capabilities"`
	SecurityCtx  SecurityContext `json:"security_context" redis:"security_context"`
}

// SecurityContext represents the security context for a UE session
type SecurityContext struct {
	KAMF                 string `json:"kamf" redis:"kamf"`
	Algorithm            string `json:"algorithm" redis:"algorithm"`
	KeySetID             string `json:"keyset_id" redis:"keyset_id"`
	NextHopChainingCount int    `json:"next_hop_chaining_count" redis:"next_hop_chaining_count"`
}

// SessionRepository defines the interface for session data operations
type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	Get(ctx context.Context, tmsi string) (*Session, error)
	Update(ctx context.Context, session *Session) error
	Delete(ctx context.Context, tmsi string) error
	QueryByIMSI(ctx context.Context, imsi string) ([]*Session, error)
	QueryByMSISDN(ctx context.Context, msisdn string) ([]*Session, error)
	QueryByMultiple(ctx context.Context, keys []string) ([]*Session, error)
	RenewTTL(ctx context.Context, tmsi string) error
}

// SessionService defines the interface for session business logic
type SessionService interface {
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, tmsi string) (*Session, error)
	UpdateSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, tmsi string) error
	QuerySessions(ctx context.Context, imsi, msisdn string) ([]*Session, error)
	RenewSession(ctx context.Context, tmsi string) error
}

// Validation errors
var (
	ErrInvalidTMSI     = &ValidationError{Field: "tmsi", Message: "TMSI is required and must be valid"}
	ErrInvalidIMSI     = &ValidationError{Field: "imsi", Message: "IMSI is required and must be valid"}
	ErrInvalidMSISDN   = &ValidationError{Field: "msisdn", Message: "MSISDN is required and must be valid"}
	ErrSessionNotFound = &NotFoundError{Resource: "session"}
	ErrSessionExpired  = &ExpiredError{Resource: "session"}
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NotFoundError represents a not found error
type NotFoundError struct {
	Resource string `json:"resource"`
}

func (e *NotFoundError) Error() string {
	return e.Resource + " not found"
}

// ExpiredError represents an expired resource error
type ExpiredError struct {
	Resource string `json:"resource"`
}

func (e *ExpiredError) Error() string {
	return e.Resource + " has expired"
}

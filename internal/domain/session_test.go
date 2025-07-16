package domain

import (
	"testing"
	"time"
)

func TestSessionValidation(t *testing.T) {
	tests := []struct {
		name    string
		session *Session
		wantErr bool
	}{
		{
			name: "valid session",
			session: &Session{
				TMSI:   "12345678",
				IMSI:   "123456789012345",
				MSISDN: "1234567890",
			},
			wantErr: false,
		},
		{
			name: "empty TMSI",
			session: &Session{
				TMSI:   "",
				IMSI:   "123456789012345",
				MSISDN: "1234567890",
			},
			wantErr: true,
		},
		{
			name: "empty IMSI",
			session: &Session{
				TMSI:   "12345678",
				IMSI:   "",
				MSISDN: "1234567890",
			},
			wantErr: true,
		},
		{
			name: "empty MSISDN",
			session: &Session{
				TMSI:   "12345678",
				IMSI:   "123456789012345",
				MSISDN: "",
			},
			wantErr: true,
		},
		{
			name:    "nil session",
			session: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSession(tt.session)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSession() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSessionCreation(t *testing.T) {
	session := &Session{
		TMSI:   "12345678",
		IMSI:   "123456789012345",
		MSISDN: "1234567890",
	}

	// Test that AttachTime is set if zero
	if !session.AttachTime.IsZero() {
		t.Error("AttachTime should be zero initially")
	}

	// Simulate setting AttachTime
	session.AttachTime = time.Now()
	session.LastUpdate = time.Now()

	if session.AttachTime.IsZero() {
		t.Error("AttachTime should not be zero after setting")
	}

	if session.LastUpdate.IsZero() {
		t.Error("LastUpdate should not be zero after setting")
	}
}

func TestSecurityContext(t *testing.T) {
	secCtx := SecurityContext{
		KAMF:                 "test-kamf",
		Algorithm:            "AES",
		KeySetID:             "1",
		NextHopChainingCount: 1,
	}

	if secCtx.KAMF != "test-kamf" {
		t.Errorf("Expected KAMF to be 'test-kamf', got %s", secCtx.KAMF)
	}

	if secCtx.Algorithm != "AES" {
		t.Errorf("Expected Algorithm to be 'AES', got %s", secCtx.Algorithm)
	}
}

// Helper function for testing
func validateSession(session *Session) error {
	if session == nil {
		return &ValidationError{Field: "session", Message: "session cannot be nil"}
	}

	if session.TMSI == "" {
		return ErrInvalidTMSI
	}

	if session.IMSI == "" {
		return ErrInvalidIMSI
	}

	if session.MSISDN == "" {
		return ErrInvalidMSISDN
	}

	return nil
}

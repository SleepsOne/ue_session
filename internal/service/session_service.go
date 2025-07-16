package service

import (
	"context"
	"fmt"

	"sessionmgr/internal/domain"
)

// SessionService implements domain.SessionService
type SessionService struct {
	repo domain.SessionRepository
}

// NewSessionService creates a new session service
func NewSessionService(repo domain.SessionRepository) *SessionService {
	return &SessionService{
		repo: repo,
	}
}

// CreateSession creates a new session with business logic validation
func (s *SessionService) CreateSession(ctx context.Context, session *domain.Session) error {
	// Business logic validation
	if err := s.validateSessionForCreation(session); err != nil {
		return err
	}

	// Check if session already exists
	existingSession, err := s.repo.Get(ctx, session.TMSI)
	if err == nil && existingSession != nil {
		return fmt.Errorf("session with TMSI %s already exists", session.TMSI)
	}

	// Set default values
	if session.UEState == "" {
		session.UEState = "REGISTERED"
	}

	if session.Capabilities == nil {
		session.Capabilities = []string{}
	}

	// Create session
	return s.repo.Create(ctx, session)
}

// GetSession retrieves a session by TMSI
func (s *SessionService) GetSession(ctx context.Context, tmsi string) (*domain.Session, error) {
	if tmsi == "" {
		return nil, domain.ErrInvalidTMSI
	}

	session, err := s.repo.Get(ctx, tmsi)
	if err != nil {
		return nil, err
	}

	// Check if session is expired (additional business logic)
	if s.isSessionExpired(session) {
		// Clean up expired session
		go s.cleanupExpiredSession(tmsi)
		return nil, domain.ErrSessionExpired
	}

	return session, nil
}

// UpdateSession updates an existing session
func (s *SessionService) UpdateSession(ctx context.Context, session *domain.Session) error {
	// Business logic validation
	if err := s.validateSessionForUpdate(session); err != nil {
		return err
	}

	// Check if session exists
	existingSession, err := s.repo.Get(ctx, session.TMSI)
	if err != nil {
		return err
	}

	// Preserve some fields that shouldn't be updated
	session.AttachTime = existingSession.AttachTime

	// Update session
	return s.repo.Update(ctx, session)
}

// DeleteSession deletes a session
func (s *SessionService) DeleteSession(ctx context.Context, tmsi string) error {
	if tmsi == "" {
		return domain.ErrInvalidTMSI
	}

	// Check if session exists
	_, err := s.repo.Get(ctx, tmsi)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, tmsi)
}

// QuerySessions queries sessions by IMSI and/or MSISDN
func (s *SessionService) QuerySessions(ctx context.Context, imsi, msisdn string) ([]*domain.Session, error) {
	var sessions []*domain.Session
	var err error

	// Query by IMSI if provided
	if imsi != "" {
		sessions, err = s.repo.QueryByIMSI(ctx, imsi)
		if err != nil {
			return nil, err
		}
	}

	// Query by MSISDN if provided
	if msisdn != "" {
		msisdnSessions, err := s.repo.QueryByMSISDN(ctx, msisdn)
		if err != nil {
			return nil, err
		}

		// Merge results if both IMSI and MSISDN are provided
		if imsi != "" {
			sessions = s.mergeSessions(sessions, msisdnSessions)
		} else {
			sessions = msisdnSessions
		}
	}

	// Filter out expired sessions
	activeSessions := s.filterActiveSessions(sessions)

	return activeSessions, nil
}

// RenewSession renews the TTL for a session
func (s *SessionService) RenewSession(ctx context.Context, tmsi string) error {
	if tmsi == "" {
		return domain.ErrInvalidTMSI
	}

	// Check if session exists
	_, err := s.repo.Get(ctx, tmsi)
	if err != nil {
		return err
	}

	return s.repo.RenewTTL(ctx, tmsi)
}

// validateSessionForCreation validates session for creation
func (s *SessionService) validateSessionForCreation(session *domain.Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	if session.TMSI == "" {
		return domain.ErrInvalidTMSI
	}

	if session.IMSI == "" {
		return domain.ErrInvalidIMSI
	}

	if session.MSISDN == "" {
		return domain.ErrInvalidMSISDN
	}

	// Additional business logic validation
	if len(session.TMSI) < 4 {
		return fmt.Errorf("TMSI must be at least 4 characters long")
	}

	if len(session.IMSI) < 14 {
		return fmt.Errorf("IMSI must be at least 14 characters long")
	}

	if len(session.MSISDN) < 10 {
		return fmt.Errorf("MSISDN must be at least 10 characters long")
	}

	return nil
}

// validateSessionForUpdate validates session for update
func (s *SessionService) validateSessionForUpdate(session *domain.Session) error {
	if session == nil {
		return fmt.Errorf("session cannot be nil")
	}

	if session.TMSI == "" {
		return domain.ErrInvalidTMSI
	}

	if session.IMSI == "" {
		return domain.ErrInvalidIMSI
	}

	if session.MSISDN == "" {
		return domain.ErrInvalidMSISDN
	}

	return nil
}

// isSessionExpired checks if a session is expired based on business rules
func (s *SessionService) isSessionExpired(session *domain.Session) bool {
	// For now, we rely on Redis TTL
	// In a more complex scenario, you might have additional business rules
	return false
}

// cleanupExpiredSession cleans up an expired session
func (s *SessionService) cleanupExpiredSession(tmsi string) {
	ctx := context.Background()
	s.repo.Delete(ctx, tmsi)
}

// mergeSessions merges two session slices and removes duplicates
func (s *SessionService) mergeSessions(sessions1, sessions2 []*domain.Session) []*domain.Session {
	// Create a map to track unique sessions by TMSI
	sessionMap := make(map[string]*domain.Session)

	// Add sessions from first slice
	for _, session := range sessions1 {
		sessionMap[session.TMSI] = session
	}

	// Add sessions from second slice
	for _, session := range sessions2 {
		sessionMap[session.TMSI] = session
	}

	// Convert map back to slice
	var mergedSessions []*domain.Session
	for _, session := range sessionMap {
		mergedSessions = append(mergedSessions, session)
	}

	return mergedSessions
}

// filterActiveSessions filters out expired sessions
func (s *SessionService) filterActiveSessions(sessions []*domain.Session) []*domain.Session {
	var activeSessions []*domain.Session

	for _, session := range sessions {
		if !s.isSessionExpired(session) {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions
}

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"sessionmgr/internal/config"
	"sessionmgr/internal/database"
	"sessionmgr/internal/domain"

	"github.com/go-redis/redis/v8"
)

// SessionRepository implements domain.SessionRepository
type SessionRepository struct {
	client *redis.Client
	config config.SessionConfig
	keys   *database.RedisKeys
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(client *redis.Client, config config.SessionConfig) *SessionRepository {
	return &SessionRepository{
		client: client,
		config: config,
		keys:   database.Keys,
	}
}

// Create creates a new session
func (r *SessionRepository) Create(ctx context.Context, session *domain.Session) error {
	// Validate session
	if err := r.validateSession(session); err != nil {
		return err
	}

	// Set current time if not set
	if session.AttachTime.IsZero() {
		session.AttachTime = time.Now()
	}
	session.LastUpdate = time.Now()

	// Serialize session to JSON
	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Store session data
	sessionKey := r.keys.SessionKey(session.TMSI)
	pipe.Set(ctx, sessionKey, sessionData, r.config.DefaultTTL)

	// Add to IMSI index
	imsiIndexKey := r.keys.IMSIIndexKey(session.IMSI)
	pipe.SAdd(ctx, imsiIndexKey, session.TMSI)
	pipe.Expire(ctx, imsiIndexKey, r.config.DefaultTTL)

	// Add to MSISDN index
	msisdnIndexKey := r.keys.MSISDNIndexKey(session.MSISDN)
	pipe.SAdd(ctx, msisdnIndexKey, session.TMSI)
	pipe.Expire(ctx, msisdnIndexKey, r.config.DefaultTTL)

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// Get retrieves a session by TMSI
func (r *SessionRepository) Get(ctx context.Context, tmsi string) (*domain.Session, error) {
	if tmsi == "" {
		return nil, domain.ErrInvalidTMSI
	}

	sessionKey := r.keys.SessionKey(tmsi)
	sessionData, err := r.client.Get(ctx, sessionKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session domain.Session
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Renew TTL on successful get
	if err := r.RenewTTL(ctx, tmsi); err != nil {
		// Log error but don't fail the get operation
		fmt.Printf("Failed to renew TTL for session %s: %v\n", tmsi, err)
	}

	return &session, nil
}

// Update updates an existing session
func (r *SessionRepository) Update(ctx context.Context, session *domain.Session) error {
	// Validate session
	if err := r.validateSession(session); err != nil {
		return err
	}

	// Check if session exists
	existingSession, err := r.Get(ctx, session.TMSI)
	if err != nil {
		return err
	}

	// Update last update time
	session.LastUpdate = time.Now()
	session.AttachTime = existingSession.AttachTime // Preserve original attach time

	// Serialize session to JSON
	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Update session data
	sessionKey := r.keys.SessionKey(session.TMSI)
	pipe.Set(ctx, sessionKey, sessionData, r.config.DefaultTTL)

	// Update IMSI index if IMSI changed
	if existingSession.IMSI != session.IMSI {
		oldIMSIKey := r.keys.IMSIIndexKey(existingSession.IMSI)
		newIMSIKey := r.keys.IMSIIndexKey(session.IMSI)
		pipe.SRem(ctx, oldIMSIKey, session.TMSI)
		pipe.SAdd(ctx, newIMSIKey, session.TMSI)
		pipe.Expire(ctx, newIMSIKey, r.config.DefaultTTL)
	}

	// Update MSISDN index if MSISDN changed
	if existingSession.MSISDN != session.MSISDN {
		oldMSISDNKey := r.keys.MSISDNIndexKey(existingSession.MSISDN)
		newMSISDNKey := r.keys.MSISDNIndexKey(session.MSISDN)
		pipe.SRem(ctx, oldMSISDNKey, session.TMSI)
		pipe.SAdd(ctx, newMSISDNKey, session.TMSI)
		pipe.Expire(ctx, newMSISDNKey, r.config.DefaultTTL)
	}

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// Delete deletes a session
func (r *SessionRepository) Delete(ctx context.Context, tmsi string) error {
	if tmsi == "" {
		return domain.ErrInvalidTMSI
	}

	// Get session to remove from indexes
	session, err := r.Get(ctx, tmsi)
	if err != nil {
		return err
	}

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Remove session data
	sessionKey := r.keys.SessionKey(tmsi)
	pipe.Del(ctx, sessionKey)

	// Remove from IMSI index
	imsiIndexKey := r.keys.IMSIIndexKey(session.IMSI)
	pipe.SRem(ctx, imsiIndexKey, tmsi)

	// Remove from MSISDN index
	msisdnIndexKey := r.keys.MSISDNIndexKey(session.MSISDN)
	pipe.SRem(ctx, msisdnIndexKey, tmsi)

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// QueryByIMSI queries sessions by IMSI
func (r *SessionRepository) QueryByIMSI(ctx context.Context, imsi string) ([]*domain.Session, error) {
	if imsi == "" {
		return nil, domain.ErrInvalidIMSI
	}

	imsiIndexKey := r.keys.IMSIIndexKey(imsi)
	tmsiList, err := r.client.SMembers(ctx, imsiIndexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to query by IMSI: %w", err)
	}

	if len(tmsiList) == 0 {
		return []*domain.Session{}, nil
	}

	return r.QueryByMultiple(ctx, tmsiList)
}

// QueryByMSISDN queries sessions by MSISDN
func (r *SessionRepository) QueryByMSISDN(ctx context.Context, msisdn string) ([]*domain.Session, error) {
	if msisdn == "" {
		return nil, domain.ErrInvalidMSISDN
	}

	msisdnIndexKey := r.keys.MSISDNIndexKey(msisdn)
	tmsiList, err := r.client.SMembers(ctx, msisdnIndexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to query by MSISDN: %w", err)
	}

	if len(tmsiList) == 0 {
		return []*domain.Session{}, nil
	}

	return r.QueryByMultiple(ctx, tmsiList)
}

// QueryByMultiple queries sessions by multiple TMSI values
func (r *SessionRepository) QueryByMultiple(ctx context.Context, tmsiList []string) ([]*domain.Session, error) {
	if len(tmsiList) == 0 {
		return []*domain.Session{}, nil
	}

	// Use pipeline to get multiple sessions
	pipe := r.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(tmsiList))

	for i, tmsi := range tmsiList {
		sessionKey := r.keys.SessionKey(tmsi)
		cmds[i] = pipe.Get(ctx, sessionKey)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to query multiple sessions: %w", err)
	}

	var sessions []*domain.Session
	for i, cmd := range cmds {
		if cmd.Err() == redis.Nil {
			// Session expired, remove from index
			go r.cleanupExpiredIndex(tmsiList[i])
			continue
		}

		if cmd.Err() != nil {
			return nil, fmt.Errorf("failed to get session %s: %w", tmsiList[i], cmd.Err())
		}

		var session domain.Session
		if err := json.Unmarshal([]byte(cmd.Val()), &session); err != nil {
			return nil, fmt.Errorf("failed to unmarshal session %s: %w", tmsiList[i], err)
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// RenewTTL renews the TTL for a session
func (r *SessionRepository) RenewTTL(ctx context.Context, tmsi string) error {
	if tmsi == "" {
		return domain.ErrInvalidTMSI
	}

	// Get session to update indexes
	session, err := r.Get(ctx, tmsi)
	if err != nil {
		return err
	}

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Renew session TTL
	sessionKey := r.keys.SessionKey(tmsi)
	pipe.Expire(ctx, sessionKey, r.config.DefaultTTL)

	// Renew IMSI index TTL
	imsiIndexKey := r.keys.IMSIIndexKey(session.IMSI)
	pipe.Expire(ctx, imsiIndexKey, r.config.DefaultTTL)

	// Renew MSISDN index TTL
	msisdnIndexKey := r.keys.MSISDNIndexKey(session.MSISDN)
	pipe.Expire(ctx, msisdnIndexKey, r.config.DefaultTTL)

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to renew TTL: %w", err)
	}

	return nil
}

// validateSession validates session data
func (r *SessionRepository) validateSession(session *domain.Session) error {
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

// cleanupExpiredIndex removes expired TMSI from indexes
func (r *SessionRepository) cleanupExpiredIndex(tmsi string) {
	ctx := context.Background()

	// This is a best-effort cleanup, so we don't return errors
	// In a production environment, you might want to implement a more robust cleanup mechanism

	// Get session to find indexes (this might fail if session is already gone)
	session, err := r.Get(ctx, tmsi)
	if err != nil {
		return
	}

	pipe := r.client.Pipeline()

	// Remove from IMSI index
	imsiIndexKey := r.keys.IMSIIndexKey(session.IMSI)
	pipe.SRem(ctx, imsiIndexKey, tmsi)

	// Remove from MSISDN index
	msisdnIndexKey := r.keys.MSISDNIndexKey(session.MSISDN)
	pipe.SRem(ctx, msisdnIndexKey, tmsi)

	pipe.Exec(ctx)
}

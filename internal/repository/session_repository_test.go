package repository

import (
	"context"
	"testing"
	"time"

	"sessionmgr/internal/config"
	"sessionmgr/internal/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Test connection
	ctx := context.Background()
	err = client.Ping(ctx).Err()
	require.NoError(t, err)

	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return client, cleanup
}

func TestSessionRepository_Create(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := config.SessionConfig{
		DefaultTTL: 30 * time.Minute,
		MaxTTL:     24 * time.Hour,
		MinTTL:     1 * time.Minute,
	}

	repo := NewSessionRepository(client, cfg)

	ctx := context.Background()
	session := &domain.Session{
		TMSI:         "12345678",
		IMSI:         "123456789012345",
		MSISDN:       "1234567890",
		GNBID:        "gNB001",
		TAI:          "TAI001",
		UEState:      "REGISTERED",
		Capabilities: []string{"5G", "4G"},
		SecurityCtx: domain.SecurityContext{
			KAMF:                 "test-kamf",
			Algorithm:            "AES",
			KeySetID:             "1",
			NextHopChainingCount: 1,
		},
	}

	// Test successful creation
	err := repo.Create(ctx, session)
	assert.NoError(t, err)

	// Verify session was created
	createdSession, err := repo.Get(ctx, session.TMSI)
	assert.NoError(t, err)
	assert.Equal(t, session.TMSI, createdSession.TMSI)
	assert.Equal(t, session.IMSI, createdSession.IMSI)
	assert.Equal(t, session.MSISDN, createdSession.MSISDN)

	// Test duplicate creation
	err = repo.Create(ctx, session)
	assert.Error(t, err)
}

func TestSessionRepository_Get(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := config.SessionConfig{
		DefaultTTL: 30 * time.Minute,
		MaxTTL:     24 * time.Hour,
		MinTTL:     1 * time.Minute,
	}

	repo := NewSessionRepository(client, cfg)
	ctx := context.Background()

	// Test getting non-existent session
	_, err := repo.Get(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, domain.ErrSessionNotFound, err)

	// Create a session
	session := &domain.Session{
		TMSI:   "12345678",
		IMSI:   "123456789012345",
		MSISDN: "1234567890",
	}

	err = repo.Create(ctx, session)
	assert.NoError(t, err)

	// Test getting existing session
	retrievedSession, err := repo.Get(ctx, session.TMSI)
	assert.NoError(t, err)
	assert.Equal(t, session.TMSI, retrievedSession.TMSI)
	assert.Equal(t, session.IMSI, retrievedSession.IMSI)
	assert.Equal(t, session.MSISDN, retrievedSession.MSISDN)
}

func TestSessionRepository_Update(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := config.SessionConfig{
		DefaultTTL: 30 * time.Minute,
		MaxTTL:     24 * time.Hour,
		MinTTL:     1 * time.Minute,
	}

	repo := NewSessionRepository(client, cfg)
	ctx := context.Background()

	// Create a session
	session := &domain.Session{
		TMSI:   "12345678",
		IMSI:   "123456789012345",
		MSISDN: "1234567890",
		GNBID:  "gNB001",
	}

	err := repo.Create(ctx, session)
	assert.NoError(t, err)

	// Update session
	session.GNBID = "gNB002"
	session.TAI = "TAI002"

	err = repo.Update(ctx, session)
	assert.NoError(t, err)

	// Verify update
	updatedSession, err := repo.Get(ctx, session.TMSI)
	assert.NoError(t, err)
	assert.Equal(t, "gNB002", updatedSession.GNBID)
	assert.Equal(t, "TAI002", updatedSession.TAI)
}

func TestSessionRepository_Delete(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := config.SessionConfig{
		DefaultTTL: 30 * time.Minute,
		MaxTTL:     24 * time.Hour,
		MinTTL:     1 * time.Minute,
	}

	repo := NewSessionRepository(client, cfg)
	ctx := context.Background()

	// Create a session
	session := &domain.Session{
		TMSI:   "12345678",
		IMSI:   "123456789012345",
		MSISDN: "1234567890",
	}

	err := repo.Create(ctx, session)
	assert.NoError(t, err)

	// Delete session
	err = repo.Delete(ctx, session.TMSI)
	assert.NoError(t, err)

	// Verify deletion
	_, err = repo.Get(ctx, session.TMSI)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrSessionNotFound, err)
}

func TestSessionRepository_QueryByIMSI(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := config.SessionConfig{
		DefaultTTL: 30 * time.Minute,
		MaxTTL:     24 * time.Hour,
		MinTTL:     1 * time.Minute,
	}

	repo := NewSessionRepository(client, cfg)
	ctx := context.Background()

	// Create multiple sessions with same IMSI
	imsi := "123456789012345"
	session1 := &domain.Session{
		TMSI:   "12345678",
		IMSI:   imsi,
		MSISDN: "1234567890",
	}
	session2 := &domain.Session{
		TMSI:   "87654321",
		IMSI:   imsi,
		MSISDN: "0987654321",
	}

	err := repo.Create(ctx, session1)
	assert.NoError(t, err)
	err = repo.Create(ctx, session2)
	assert.NoError(t, err)

	// Query by IMSI
	sessions, err := repo.QueryByIMSI(ctx, imsi)
	assert.NoError(t, err)
	assert.Len(t, sessions, 2)

	// Verify both sessions are returned
	tmsiSet := make(map[string]bool)
	for _, s := range sessions {
		tmsiSet[s.TMSI] = true
	}
	assert.True(t, tmsiSet["12345678"])
	assert.True(t, tmsiSet["87654321"])
}

func TestSessionRepository_QueryByMSISDN(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := config.SessionConfig{
		DefaultTTL: 30 * time.Minute,
		MaxTTL:     24 * time.Hour,
		MinTTL:     1 * time.Minute,
	}

	repo := NewSessionRepository(client, cfg)
	ctx := context.Background()

	// Create a session
	session := &domain.Session{
		TMSI:   "12345678",
		IMSI:   "123456789012345",
		MSISDN: "1234567890",
	}

	err := repo.Create(ctx, session)
	assert.NoError(t, err)

	// Query by MSISDN
	sessions, err := repo.QueryByMSISDN(ctx, session.MSISDN)
	assert.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, session.TMSI, sessions[0].TMSI)
}

func TestSessionRepository_RenewTTL(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := config.SessionConfig{
		DefaultTTL: 30 * time.Minute,
		MaxTTL:     24 * time.Hour,
		MinTTL:     1 * time.Minute,
	}

	repo := NewSessionRepository(client, cfg)
	ctx := context.Background()

	// Create a session
	session := &domain.Session{
		TMSI:   "12345678",
		IMSI:   "123456789012345",
		MSISDN: "1234567890",
	}

	err := repo.Create(ctx, session)
	assert.NoError(t, err)

	// Renew TTL
	err = repo.RenewTTL(ctx, session.TMSI)
	assert.NoError(t, err)

	// Verify session still exists
	_, err = repo.Get(ctx, session.TMSI)
	assert.NoError(t, err)
}

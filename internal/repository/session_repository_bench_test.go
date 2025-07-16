package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"sessionmgr/internal/config"
	"sessionmgr/internal/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func setupBenchmarkRedis(b *testing.B) (*redis.Client, func()) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatal(err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Test connection
	ctx := context.Background()
	err = client.Ping(ctx).Err()
	if err != nil {
		b.Fatal(err)
	}

	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return client, cleanup
}

func BenchmarkSessionRepository_Create(b *testing.B) {
	client, cleanup := setupBenchmarkRedis(b)
	defer cleanup()

	cfg := config.SessionConfig{
		DefaultTTL: 30 * time.Minute,
		MaxTTL:     24 * time.Hour,
		MinTTL:     1 * time.Minute,
	}

	repo := NewSessionRepository(client, cfg)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			session := &domain.Session{
				TMSI:    fmt.Sprintf("TMSI%08d", i),
				IMSI:    fmt.Sprintf("IMSI%015d", i),
				MSISDN:  fmt.Sprintf("MSISDN%010d", i),
				GNBID:   fmt.Sprintf("gNB%03d", i%100),
				TAI:     fmt.Sprintf("TAI%03d", i%100),
				UEState: "REGISTERED",
			}
			err := repo.Create(ctx, session)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkSessionRepository_Get(b *testing.B) {
	client, cleanup := setupBenchmarkRedis(b)
	defer cleanup()

	cfg := config.SessionConfig{
		DefaultTTL: 30 * time.Minute,
		MaxTTL:     24 * time.Hour,
		MinTTL:     1 * time.Minute,
	}

	repo := NewSessionRepository(client, cfg)
	ctx := context.Background()

	// Pre-create sessions
	sessions := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		session := &domain.Session{
			TMSI:   fmt.Sprintf("TMSI%08d", i),
			IMSI:   fmt.Sprintf("IMSI%015d", i),
			MSISDN: fmt.Sprintf("MSISDN%010d", i),
		}
		err := repo.Create(ctx, session)
		if err != nil {
			b.Fatal(err)
		}
		sessions[i] = session.TMSI
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			tmsi := sessions[i%len(sessions)]
			_, err := repo.Get(ctx, tmsi)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkSessionRepository_QueryByIMSI(b *testing.B) {
	client, cleanup := setupBenchmarkRedis(b)
	defer cleanup()

	cfg := config.SessionConfig{
		DefaultTTL: 30 * time.Minute,
		MaxTTL:     24 * time.Hour,
		MinTTL:     1 * time.Minute,
	}

	repo := NewSessionRepository(client, cfg)
	ctx := context.Background()

	// Pre-create sessions with same IMSI
	imsi := "123456789012345"
	for i := 0; i < 100; i++ {
		session := &domain.Session{
			TMSI:   fmt.Sprintf("TMSI%08d", i),
			IMSI:   imsi,
			MSISDN: fmt.Sprintf("MSISDN%010d", i),
		}
		err := repo.Create(ctx, session)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := repo.QueryByIMSI(ctx, imsi)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkSessionRepository_Update(b *testing.B) {
	client, cleanup := setupBenchmarkRedis(b)
	defer cleanup()

	cfg := config.SessionConfig{
		DefaultTTL: 30 * time.Minute,
		MaxTTL:     24 * time.Hour,
		MinTTL:     1 * time.Minute,
	}

	repo := NewSessionRepository(client, cfg)
	ctx := context.Background()

	// Pre-create sessions
	sessions := make([]*domain.Session, 1000)
	for i := 0; i < 1000; i++ {
		session := &domain.Session{
			TMSI:   fmt.Sprintf("TMSI%08d", i),
			IMSI:   fmt.Sprintf("IMSI%015d", i),
			MSISDN: fmt.Sprintf("MSISDN%010d", i),
			GNBID:  fmt.Sprintf("gNB%03d", i%100),
		}
		err := repo.Create(ctx, session)
		if err != nil {
			b.Fatal(err)
		}
		sessions[i] = session
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			session := sessions[i%len(sessions)]
			session.GNBID = fmt.Sprintf("gNB%03d", i%200)
			err := repo.Update(ctx, session)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkSessionRepository_Delete(b *testing.B) {
	client, cleanup := setupBenchmarkRedis(b)
	defer cleanup()

	cfg := config.SessionConfig{
		DefaultTTL: 30 * time.Minute,
		MaxTTL:     24 * time.Hour,
		MinTTL:     1 * time.Minute,
	}

	repo := NewSessionRepository(client, cfg)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Create a session first
			session := &domain.Session{
				TMSI:   fmt.Sprintf("TMSI%08d", i),
				IMSI:   fmt.Sprintf("IMSI%015d", i),
				MSISDN: fmt.Sprintf("MSISDN%010d", i),
			}
			err := repo.Create(ctx, session)
			if err != nil {
				b.Fatal(err)
			}

			// Then delete it
			err = repo.Delete(ctx, session.TMSI)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

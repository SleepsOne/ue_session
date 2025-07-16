package database

import (
	"context"
	"fmt"
	"time"

	"sessionmgr/internal/config"

	"github.com/go-redis/redis/v8"
)

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}

// RedisKeys defines Redis key patterns
type RedisKeys struct{}

// SessionKey returns the Redis key for a session
func (rk *RedisKeys) SessionKey(tmsi string) string {
	return fmt.Sprintf("sess:%s", tmsi)
}

// IMSIIndexKey returns the Redis key for IMSI index
func (rk *RedisKeys) IMSIIndexKey(imsi string) string {
	return fmt.Sprintf("idx:imsi:%s", imsi)
}

// MSISDNIndexKey returns the Redis key for MSISDN index
func (rk *RedisKeys) MSISDNIndexKey(msisdn string) string {
	return fmt.Sprintf("idx:msisdn:%s", msisdn)
}

// Global keys instance
var Keys = &RedisKeys{}

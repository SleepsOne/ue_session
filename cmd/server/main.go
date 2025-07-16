package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sessionmgr/internal/config"
	"sessionmgr/internal/database"
	"sessionmgr/internal/handler"
	"sessionmgr/internal/repository"
	"sessionmgr/internal/service"

	"github.com/gin-gonic/gin"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Redis connection
	redisClient, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize repository
	sessionRepo := repository.NewSessionRepository(redisClient, cfg.Session)

	// Initialize service
	sessionService := service.NewSessionService(sessionRepo)

	// Initialize handlers
	sessionHandler := handler.NewSessionHandler(sessionService)

	// Setup Gin router
	router := gin.Default()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Setup routes
	setupRoutes(router, sessionHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

func setupRoutes(router *gin.Engine, sessionHandler *handler.SessionHandler) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"version":   Version,
			"buildTime": BuildTime,
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		sessions := api.Group("/sessions")
		{
			sessions.POST("", sessionHandler.Create)
			sessions.GET("/:id", sessionHandler.Get)
			sessions.PUT("/:id", sessionHandler.Update)
			sessions.DELETE("/:id", sessionHandler.Delete)
			sessions.GET("", sessionHandler.Query)
			sessions.POST("/:id/renew", sessionHandler.Renew)
		}
	}
}

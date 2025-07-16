package handler

import (
	"net/http"

	"sessionmgr/internal/domain"

	"github.com/gin-gonic/gin"
)

// SessionHandler handles HTTP requests for session operations
type SessionHandler struct {
	service domain.SessionService
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(service domain.SessionService) *SessionHandler {
	return &SessionHandler{
		service: service,
	}
}

// Create handles POST /sessions
func (h *SessionHandler) Create(c *gin.Context) {
	var session domain.Session
	if err := c.ShouldBindJSON(&session); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Extract TMSI from path if provided
	if tmsi := c.Param("tmsi"); tmsi != "" {
		session.TMSI = tmsi
	}

	if err := h.service.CreateSession(c.Request.Context(), &session); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Session created successfully",
		"session": session,
	})
}

// Get handles GET /sessions/:id
func (h *SessionHandler) Get(c *gin.Context) {
	tmsi := c.Param("id")
	if tmsi == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "TMSI is required",
		})
		return
	}

	session, err := h.service.GetSession(c.Request.Context(), tmsi)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session": session,
	})
}

// Update handles PUT /sessions/:id
func (h *SessionHandler) Update(c *gin.Context) {
	tmsi := c.Param("id")
	if tmsi == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "TMSI is required",
		})
		return
	}

	var session domain.Session
	if err := c.ShouldBindJSON(&session); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Ensure TMSI in path matches TMSI in body
	session.TMSI = tmsi

	if err := h.service.UpdateSession(c.Request.Context(), &session); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Session updated successfully",
		"session": session,
	})
}

// Delete handles DELETE /sessions/:id
func (h *SessionHandler) Delete(c *gin.Context) {
	tmsi := c.Param("id")
	if tmsi == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "TMSI is required",
		})
		return
	}

	if err := h.service.DeleteSession(c.Request.Context(), tmsi); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Session deleted successfully",
	})
}

// Query handles GET /sessions with query parameters
func (h *SessionHandler) Query(c *gin.Context) {
	imsi := c.Query("imsi")
	msisdn := c.Query("msisdn")

	// At least one query parameter is required
	if imsi == "" && msisdn == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "At least one query parameter (imsi or msisdn) is required",
		})
		return
	}

	sessions, err := h.service.QuerySessions(c.Request.Context(), imsi, msisdn)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// Renew handles POST /sessions/:id/renew
func (h *SessionHandler) Renew(c *gin.Context) {
	tmsi := c.Param("id")
	if tmsi == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "TMSI is required",
		})
		return
	}

	if err := h.service.RenewSession(c.Request.Context(), tmsi); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Session TTL renewed successfully",
	})
}

// handleError handles different types of errors and returns appropriate HTTP responses
func (h *SessionHandler) handleError(c *gin.Context, err error) {
	switch {
	case err == domain.ErrSessionNotFound:
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Session not found",
		})
	case err == domain.ErrSessionExpired:
		c.JSON(http.StatusGone, gin.H{
			"error": "Session has expired",
		})
	case err == domain.ErrInvalidTMSI:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid TMSI",
		})
	case err == domain.ErrInvalidIMSI:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid IMSI",
		})
	case err == domain.ErrInvalidMSISDN:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid MSISDN",
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
	}
}

// Health handles GET /health
func (h *SessionHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "session-manager",
	})
}

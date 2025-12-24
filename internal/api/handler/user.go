package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/api/service"
	"github.com/kedr891/cs-parser/internal/domain"
	"github.com/kedr891/cs-parser/internal/entity"
)

type UserHandler struct {
	service *service.UserService
	log     domain.Logger
}

func NewUserHandler(service *service.UserService, log domain.Logger) *UserHandler {
	return &UserHandler{
		service: service,
		log:     log,
	}
}

// Register - регистрация пользователя
// @Summary Register new user
// @Tags auth
// @Param request body entity.RegisterRequest true "Register request"
// @Success 201 {object} entity.LoginResponse
// @Router /api/v1/auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req entity.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("Failed to register user", "error", err)
		if err.Error() == "user with this email already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		}
		return
	}

	c.JSON(http.StatusCreated, response)
}

// Login - вход пользователя
// @Summary Login user
// @Tags auth
// @Param request body entity.LoginRequest true "Login request"
// @Success 200 {object} entity.LoginResponse
// @Router /api/v1/auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req entity.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.service.Login(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("Failed to login", "error", err)
		if err.Error() == "account is disabled" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		}
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetProfile - получить профиль текущего пользователя
// @Summary Get user profile
// @Tags users
// @Security BearerAuth
// @Success 200 {object} entity.UserResponse
// @Router /api/v1/users/me [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	profile, err := h.service.GetProfile(c.Request.Context(), id)
	if err != nil {
		h.log.Error("Failed to get profile", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// GetWatchlist - получить watchlist пользователя
// @Summary Get user watchlist
// @Tags users
// @Security BearerAuth
// @Success 200 {array} entity.WatchlistWithSkin
// @Router /api/v1/users/me/watchlist [get]
func (h *UserHandler) GetWatchlist(c *gin.Context) {
	userID := c.GetString("user_id")
	id, _ := uuid.Parse(userID)

	watchlist, err := h.service.GetWatchlist(c.Request.Context(), id)
	if err != nil {
		h.log.Error("Failed to get watchlist", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get watchlist"})
		return
	}

	c.JSON(http.StatusOK, watchlist)
}

// AddToWatchlist - добавить скин в watchlist
// @Summary Add skin to watchlist
// @Tags users
// @Security BearerAuth
// @Param skin_id path string true "Skin ID"
// @Success 201 {object} entity.Watchlist
// @Router /api/v1/users/me/watchlist/{skin_id} [post]
func (h *UserHandler) AddToWatchlist(c *gin.Context) {
	userID := c.GetString("user_id")
	skinIDStr := c.Param("skin_id")

	uid, _ := uuid.Parse(userID)
	skinID, err := uuid.Parse(skinIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid skin ID"})
		return
	}

	var body struct {
		TargetPrice  *float64 `json:"target_price"`
		NotifyOnDrop bool     `json:"notify_on_drop"`
	}
	_ = c.ShouldBindJSON(&body)

	watchlist, err := h.service.AddToWatchlist(c.Request.Context(), uid, skinID, body.TargetPrice, body.NotifyOnDrop)
	if err != nil {
		h.log.Error("Failed to add to watchlist", "error", err)
		if err.Error() == "skin already in watchlist" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to watchlist"})
		}
		return
	}

	c.JSON(http.StatusCreated, watchlist)
}

// RemoveFromWatchlist - удалить скин из watchlist
// @Summary Remove skin from watchlist
// @Tags users
// @Security BearerAuth
// @Param skin_id path string true "Skin ID"
// @Success 204
// @Router /api/v1/users/me/watchlist/{skin_id} [delete]
func (h *UserHandler) RemoveFromWatchlist(c *gin.Context) {
	userID := c.GetString("user_id")
	skinIDStr := c.Param("skin_id")

	uid, _ := uuid.Parse(userID)
	skinID, err := uuid.Parse(skinIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid skin ID"})
		return
	}

	if err := h.service.RemoveFromWatchlist(c.Request.Context(), uid, skinID); err != nil {
		h.log.Error("Failed to remove from watchlist", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove from watchlist"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetNotifications - получить уведомления пользователя
// @Summary Get user notifications
// @Tags users
// @Security BearerAuth
// @Param unread query bool false "Only unread"
// @Param limit query int false "Limit"
// @Success 200 {object} entity.NotificationListResponse
// @Router /api/v1/users/me/notifications [get]
func (h *UserHandler) GetNotifications(c *gin.Context) {
	userID := c.GetString("user_id")
	uid, _ := uuid.Parse(userID)

	unreadOnly := c.Query("unread") == "true"
	limit := 50
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	response, err := h.service.GetNotifications(c.Request.Context(), uid, unreadOnly, limit)
	if err != nil {
		h.log.Error("Failed to get notifications", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notifications"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// MarkNotificationsRead - пометить уведомления как прочитанные
// @Summary Mark notifications as read
// @Tags users
// @Security BearerAuth
// @Param request body entity.MarkReadRequest true "Notification IDs"
// @Success 200
// @Router /api/v1/users/me/notifications/read [post]
func (h *UserHandler) MarkNotificationsRead(c *gin.Context) {
	userID := c.GetString("user_id")
	uid, _ := uuid.Parse(userID)

	var req entity.MarkReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.MarkNotificationsRead(c.Request.Context(), uid, req.NotificationIDs); err != nil {
		h.log.Error("Failed to mark notifications as read", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetStats - получить статистику пользователя
// @Summary Get user stats
// @Tags users
// @Security BearerAuth
// @Success 200 {object} entity.UserStats
// @Router /api/v1/users/me/stats [get]
func (h *UserHandler) GetStats(c *gin.Context) {
	userID := c.GetString("user_id")
	uid, _ := uuid.Parse(userID)

	stats, err := h.service.GetUserStats(c.Request.Context(), uid)
	if err != nil {
		h.log.Error("Failed to get user stats", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

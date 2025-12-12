package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cs-parser/internal/entity"
	"github.com/cs-parser/pkg/logger"
	"github.com/cs-parser/pkg/postgres"
	"github.com/cs-parser/pkg/redis"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// UserHandler - обработчик для пользователей
type UserHandler struct {
	pg        *postgres.Postgres
	redis     *redis.Redis
	jwtSecret string
	log       *logger.Logger
}

// NewUserHandler - создать обработчик пользователей
func NewUserHandler(pg *postgres.Postgres, redis *redis.Redis, jwtSecret string, log *logger.Logger) *UserHandler {
	return &UserHandler{
		pg:        pg,
		redis:     redis,
		jwtSecret: jwtSecret,
		log:       log,
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

	// Проверить существование пользователя
	exists, err := h.userExistsByEmail(c.Request.Context(), req.Email)
	if err != nil {
		h.log.Error("Failed to check user existence", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	// Создать пользователя
	user, err := entity.NewUser(req.Email, req.Username, req.Password)
	if err != nil {
		h.log.Error("Failed to create user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Сохранить в БД
	if err := h.createUser(c.Request.Context(), user); err != nil {
		h.log.Error("Failed to save user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save user"})
		return
	}

	// Создать настройки по умолчанию
	settings := entity.NewUserSettings(user.ID)
	if err := h.createUserSettings(c.Request.Context(), settings); err != nil {
		h.log.Warn("Failed to create user settings", "error", err)
	}

	// Генерация JWT токена
	token, err := h.generateJWT(user)
	if err != nil {
		h.log.Error("Failed to generate token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	response := entity.LoginResponse{
		User:  user.Sanitize(),
		Token: token,
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

	// Найти пользователя
	user, err := h.getUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Проверить пароль
	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Проверить активность аккаунта
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is disabled"})
		return
	}

	// Обновить время последнего входа
	user.UpdateLastLogin()
	if err := h.updateLastLogin(c.Request.Context(), user.ID); err != nil {
		h.log.Warn("Failed to update last login", "error", err)
	}

	// Генерация JWT токена
	token, err := h.generateJWT(user)
	if err != nil {
		h.log.Error("Failed to generate token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	response := entity.LoginResponse{
		User:  user.Sanitize(),
		Token: token,
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

	user, err := h.getUserByID(c.Request.Context(), id)
	if err != nil {
		h.log.Error("Failed to get user", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user.Sanitize())
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

	watchlist, err := h.getWatchlistWithSkins(c.Request.Context(), id)
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

	// Проверить существование скина
	if _, err := h.getSkinByID(c.Request.Context(), skinID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Skin not found"})
		return
	}

	// Проверить, не добавлен ли уже
	exists, err := h.watchlistExists(c.Request.Context(), uid, skinID)
	if err != nil {
		h.log.Error("Failed to check watchlist", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Skin already in watchlist"})
		return
	}

	// Создать запись в watchlist
	watchlist := entity.NewWatchlist(uid, skinID)

	// Получить target_price из body если есть
	var body struct {
		TargetPrice  *float64 `json:"target_price"`
		NotifyOnDrop bool     `json:"notify_on_drop"`
	}
	if err := c.ShouldBindJSON(&body); err == nil {
		if body.TargetPrice != nil {
			watchlist.SetTargetPrice(*body.TargetPrice)
		}
		watchlist.NotifyOnDrop = body.NotifyOnDrop
	}

	if err := h.createWatchlist(c.Request.Context(), watchlist); err != nil {
		h.log.Error("Failed to create watchlist", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to watchlist"})
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

	if err := h.deleteWatchlist(c.Request.Context(), uid, skinID); err != nil {
		h.log.Error("Failed to remove from watchlist", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove from watchlist"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Helper methods

func (h *UserHandler) generateJWT(user *entity.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(168 * time.Hour).Unix(), // 7 days
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

func (h *UserHandler) userExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := h.pg.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists)
	return exists, err
}

func (h *UserHandler) createUser(ctx context.Context, user *entity.User) error {
	query := `
		INSERT INTO users (id, email, username, password_hash, role, is_active, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := h.pg.Pool.Exec(ctx, query,
		user.ID, user.Email, user.Username, user.PasswordHash,
		user.Role, user.IsActive, user.IsVerified,
		user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (h *UserHandler) getUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT id, email, username, password_hash, role, is_active, is_verified, last_login_at, created_at, updated_at
		FROM users WHERE email = $1
	`
	var user entity.User
	err := h.pg.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Username, &user.PasswordHash,
		&user.Role, &user.IsActive, &user.IsVerified,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return &user, err
}

func (h *UserHandler) getUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	query := `
		SELECT id, email, username, password_hash, role, is_active, is_verified, last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`
	var user entity.User
	err := h.pg.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Username, &user.PasswordHash,
		&user.Role, &user.IsActive, &user.IsVerified,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	return &user, err
}

func (h *UserHandler) updateLastLogin(ctx context.Context, userID uuid.UUID) error {
	_, err := h.pg.Pool.Exec(ctx, `UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`, userID)
	return err
}

func (h *UserHandler) createUserSettings(ctx context.Context, settings *entity.UserSettings) error {
	query := `
		INSERT INTO user_settings (
			user_id, email_notifications, push_notifications,
			price_alert_threshold, preferred_currency, notification_frequency, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := h.pg.Pool.Exec(ctx, query,
		settings.UserID, settings.EmailNotifications, settings.PushNotifications,
		settings.PriceAlertThreshold, settings.PreferredCurrency,
		settings.NotificationFrequency, settings.UpdatedAt,
	)
	return err
}

func (h *UserHandler) getWatchlistWithSkins(ctx context.Context, userID uuid.UUID) ([]entity.WatchlistWithSkin, error) {
	query := `
		SELECT 
			w.id, w.user_id, w.skin_id, w.target_price, w.notify_on_drop, w.notify_on_price, w.is_active, w.added_at, w.updated_at,
			s.id, s.market_hash_name, s.name, s.weapon, s.quality, s.rarity, s.current_price, s.currency, s.image_url
		FROM watchlist w
		JOIN skins s ON w.skin_id = s.id
		WHERE w.user_id = $1 AND w.is_active = true
		ORDER BY w.added_at DESC
	`

	rows, err := h.pg.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []entity.WatchlistWithSkin
	for rows.Next() {
		var wws entity.WatchlistWithSkin
		err := rows.Scan(
			&wws.ID, &wws.UserID, &wws.SkinID, &wws.TargetPrice, &wws.NotifyOnDrop, &wws.NotifyOnPrice, &wws.IsActive, &wws.AddedAt, &wws.UpdatedAt,
			&wws.Skin.ID, &wws.Skin.MarketHashName, &wws.Skin.Name, &wws.Skin.Weapon, &wws.Skin.Quality, &wws.Skin.Rarity,
			&wws.Skin.CurrentPrice, &wws.Skin.Currency, &wws.Skin.ImageURL,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, wws)
	}

	return result, nil
}

func (h *UserHandler) watchlistExists(ctx context.Context, userID, skinID uuid.UUID) (bool, error) {
	var exists bool
	err := h.pg.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM watchlist WHERE user_id = $1 AND skin_id = $2 AND is_active = true)`,
		userID, skinID,
	).Scan(&exists)
	return exists, err
}

func (h *UserHandler) createWatchlist(ctx context.Context, wl *entity.Watchlist) error {
	query := `
		INSERT INTO watchlist (id, user_id, skin_id, target_price, notify_on_drop, notify_on_price, is_active, added_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := h.pg.Pool.Exec(ctx, query,
		wl.ID, wl.UserID, wl.SkinID, wl.TargetPrice,
		wl.NotifyOnDrop, wl.NotifyOnPrice, wl.IsActive,
		wl.AddedAt, wl.UpdatedAt,
	)
	return err
}

func (h *UserHandler) deleteWatchlist(ctx context.Context, userID, skinID uuid.UUID) error {
	_, err := h.pg.Pool.Exec(ctx,
		`UPDATE watchlist SET is_active = false, updated_at = NOW() WHERE user_id = $1 AND skin_id = $2`,
		userID, skinID,
	)
	return err
}

func (h *UserHandler) getSkinByID(ctx context.Context, skinID uuid.UUID) (*entity.Skin, error) {
	var skin entity.Skin
	err := h.pg.Pool.QueryRow(ctx, `SELECT id FROM skins WHERE id = $1`, skinID).Scan(&skin.ID)
	return &skin, err
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

	notifications, err := h.getNotifications(c.Request.Context(), uid, unreadOnly, limit)
	if err != nil {
		h.log.Error("Failed to get notifications", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notifications"})
		return
	}

	// Подсчитать непрочитанные
	unreadCount, _ := h.getUnreadCount(c.Request.Context(), uid)

	response := entity.NotificationListResponse{
		Notifications: notifications,
		Total:         len(notifications),
		Unread:        unreadCount,
		Page:          1,
		PageSize:      limit,
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

	if err := h.markNotificationsRead(c.Request.Context(), uid, req.NotificationIDs); err != nil {
		h.log.Error("Failed to mark notifications as read", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *UserHandler) getNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit int) ([]entity.Notification, error) {
	query := `
		SELECT id, user_id, type, title, message, is_read, priority, created_at, read_at
		FROM notifications
		WHERE user_id = $1
	`

	if unreadOnly {
		query += " AND is_read = false"
	}

	query += " ORDER BY created_at DESC LIMIT $2"

	rows, err := h.pg.Pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []entity.Notification
	for rows.Next() {
		var n entity.Notification
		err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Message, &n.IsRead, &n.Priority, &n.CreatedAt, &n.ReadAt)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}

	return notifications, nil
}

func (h *UserHandler) getUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := h.pg.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`, userID).Scan(&count)
	return count, err
}

func (h *UserHandler) markNotificationsRead(ctx context.Context, userID uuid.UUID, notificationIDs []uuid.UUID) error {
	query := `
		UPDATE notifications
		SET is_read = true, read_at = NOW()
		WHERE user_id = $1 AND id = ANY($2)
	`
	_, err := h.pg.Pool.Exec(ctx, query, userID, notificationIDs)
	return err
}

package entity

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User - модель пользователя
type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	Username     string     `json:"username" db:"username"`
	PasswordHash string     `json:"-" db:"password_hash"` // не отдаём в JSON
	Role         UserRole   `json:"role" db:"role"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	IsVerified   bool       `json:"is_verified" db:"is_verified"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// UserRole - роль пользователя
type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

// NewUser - создать нового пользователя
func NewUser(email, username, password string) (*User, error) {
	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         RoleUser,
		IsActive:     true,
		IsVerified:   false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// HashPassword - захешировать пароль
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword - проверить пароль
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// UpdateLastLogin - обновить время последнего входа
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.UpdatedAt = now
}

// IsAdmin - является ли пользователь администратором
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// Sanitize - убрать чувствительные данные перед отправкой
func (u *User) Sanitize() *UserResponse {
	return &UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		Username:    u.Username,
		Role:        u.Role,
		IsActive:    u.IsActive,
		IsVerified:  u.IsVerified,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
	}
}

// UserResponse - безопасный ответ с данными пользователя
type UserResponse struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	Username    string     `json:"username"`
	Role        UserRole   `json:"role"`
	IsActive    bool       `json:"is_active"`
	IsVerified  bool       `json:"is_verified"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Watchlist - отслеживаемый скин
type Watchlist struct {
	ID            uuid.UUID `json:"id" db:"id"`
	UserID        uuid.UUID `json:"user_id" db:"user_id"`
	SkinID        uuid.UUID `json:"skin_id" db:"skin_id"`
	TargetPrice   *float64  `json:"target_price,omitempty" db:"target_price"`
	NotifyOnDrop  bool      `json:"notify_on_drop" db:"notify_on_drop"`
	NotifyOnPrice bool      `json:"notify_on_price" db:"notify_on_price"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	AddedAt       time.Time `json:"added_at" db:"added_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// NewWatchlist - создать запись в watchlist
func NewWatchlist(userID, skinID uuid.UUID) *Watchlist {
	now := time.Now()
	return &Watchlist{
		ID:           uuid.New(),
		UserID:       userID,
		SkinID:       skinID,
		NotifyOnDrop: true,
		IsActive:     true,
		AddedAt:      now,
		UpdatedAt:    now,
	}
}

// SetTargetPrice - установить целевую цену
func (w *Watchlist) SetTargetPrice(price float64) {
	w.TargetPrice = &price
	w.NotifyOnPrice = true
	w.UpdatedAt = time.Now()
}

// ShouldNotify - нужно ли отправлять уведомление
func (w *Watchlist) ShouldNotify(currentPrice, oldPrice float64) bool {
	if !w.IsActive {
		return false
	}

	// Уведомление при падении цены
	if w.NotifyOnDrop && currentPrice < oldPrice {
		return true
	}

	// Уведомление при достижении целевой цены
	if w.NotifyOnPrice && w.TargetPrice != nil && currentPrice <= *w.TargetPrice {
		return true
	}

	return false
}

// WatchlistWithSkin - watchlist с данными скина
type WatchlistWithSkin struct {
	Watchlist
	Skin Skin `json:"skin"`
}

// UserSettings - настройки пользователя
type UserSettings struct {
	UserID                uuid.UUID `json:"user_id" db:"user_id"`
	EmailNotifications    bool      `json:"email_notifications" db:"email_notifications"`
	PushNotifications     bool      `json:"push_notifications" db:"push_notifications"`
	PriceAlertThreshold   float64   `json:"price_alert_threshold" db:"price_alert_threshold"` // % изменения цены
	PreferredCurrency     string    `json:"preferred_currency" db:"preferred_currency"`
	NotificationFrequency string    `json:"notification_frequency" db:"notification_frequency"` // instant, hourly, daily
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

// NewUserSettings - создать настройки по умолчанию
func NewUserSettings(userID uuid.UUID) *UserSettings {
	return &UserSettings{
		UserID:                userID,
		EmailNotifications:    true,
		PushNotifications:     false,
		PriceAlertThreshold:   5.0,
		PreferredCurrency:     "USD",
		NotificationFrequency: "instant",
		UpdatedAt:             time.Now(),
	}
}

// RegisterRequest - запрос на регистрацию
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest - запрос на вход
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse - ответ при входе
type LoginResponse struct {
	User  *UserResponse `json:"user"`
	Token string        `json:"token"`
}

// UpdateProfileRequest - запрос на обновление профиля
type UpdateProfileRequest struct {
	Username string `json:"username" binding:"omitempty,min=3,max=50"`
	Email    string `json:"email" binding:"omitempty,email"`
}

// ChangePasswordRequest - запрос на смену пароля
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// UserStats - статистика пользователя
type UserStats struct {
	TotalWatchlist      int       `json:"total_watchlist"`
	ActiveAlerts        int       `json:"active_alerts"`
	TotalNotifications  int       `json:"total_notifications"`
	UnreadNotifications int       `json:"unread_notifications"`
	MemberSince         time.Time `json:"member_since"`
}

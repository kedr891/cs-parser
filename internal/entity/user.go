package entity

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Email        string     `json:"email" db:"email"`
	Username     string     `json:"username" db:"username"`
	PasswordHash string     `json:"-" db:"password_hash"`
	Role         UserRole   `json:"role" db:"role"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	IsVerified   bool       `json:"is_verified" db:"is_verified"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

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

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.UpdatedAt = now
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

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

func (w *Watchlist) SetTargetPrice(price float64) {
	w.TargetPrice = &price
	w.NotifyOnPrice = true
	w.UpdatedAt = time.Now()
}

func (w *Watchlist) ShouldNotify(currentPrice, oldPrice float64) bool {
	if !w.IsActive {
		return false
	}

	if w.NotifyOnDrop && currentPrice < oldPrice {
		return true
	}

	if w.NotifyOnPrice && w.TargetPrice != nil && currentPrice <= *w.TargetPrice {
		return true
	}

	return false
}

type WatchlistWithSkin struct {
	Watchlist
	Skin Skin `json:"skin"`
}

type UserSettings struct {
	UserID                uuid.UUID `json:"user_id" db:"user_id"`
	EmailNotifications    bool      `json:"email_notifications" db:"email_notifications"`
	PushNotifications     bool      `json:"push_notifications" db:"push_notifications"`
	PriceAlertThreshold   float64   `json:"price_alert_threshold" db:"price_alert_threshold"`
	PreferredCurrency     string    `json:"preferred_currency" db:"preferred_currency"`
	NotificationFrequency string    `json:"notification_frequency" db:"notification_frequency"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}

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

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	User  *UserResponse `json:"user"`
	Token string        `json:"token"`
}

type UpdateProfileRequest struct {
	Username string `json:"username" binding:"omitempty,min=3,max=50"`
	Email    string `json:"email" binding:"omitempty,email"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type UserStats struct {
	TotalWatchlist      int       `json:"total_watchlist"`
	ActiveAlerts        int       `json:"active_alerts"`
	TotalNotifications  int       `json:"total_notifications"`
	UnreadNotifications int       `json:"unread_notifications"`
	MemberSince         time.Time `json:"member_since"`
}

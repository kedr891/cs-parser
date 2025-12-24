package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	UserID    uuid.UUID              `json:"user_id" db:"user_id"`
	Type      NotificationType       `json:"type" db:"type"`
	Title     string                 `json:"title" db:"title"`
	Message   string                 `json:"message" db:"message"`
	Data      map[string]interface{} `json:"data,omitempty" db:"data"` // JSON
	IsRead    bool                   `json:"is_read" db:"is_read"`
	Priority  NotificationPriority   `json:"priority" db:"priority"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	ReadAt    *time.Time             `json:"read_at,omitempty" db:"read_at"`
}

type NotificationType string

const (
	TypePriceDrop       NotificationType = "price_drop"
	TypePriceIncrease   NotificationType = "price_increase"
	TypeTargetReached   NotificationType = "target_reached"
	TypeSkinDiscovered  NotificationType = "skin_discovered"
	TypeSystemAlert     NotificationType = "system_alert"
	TypeWatchlistUpdate NotificationType = "watchlist_update"
)

type NotificationPriority string

const (
	PriorityLow    NotificationPriority = "low"
	PriorityNormal NotificationPriority = "normal"
	PriorityHigh   NotificationPriority = "high"
	PriorityUrgent NotificationPriority = "urgent"
)

func NewNotification(userID uuid.UUID, notifType NotificationType, title, message string) *Notification {
	return &Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Message:   message,
		IsRead:    false,
		Priority:  PriorityNormal,
		Data:      make(map[string]interface{}),
		CreatedAt: time.Now(),
	}
}

func (n *Notification) MarkAsRead() {
	if !n.IsRead {
		n.IsRead = true
		now := time.Now()
		n.ReadAt = &now
	}
}

func (n *Notification) SetPriority(priority NotificationPriority) {
	n.Priority = priority
}

func (n *Notification) AddData(key string, value interface{}) {
	if n.Data == nil {
		n.Data = make(map[string]interface{})
	}
	n.Data[key] = value
}

type PriceAlertNotification struct {
	UserID           uuid.UUID        `json:"user_id"`
	SkinID           uuid.UUID        `json:"skin_id"`
	MarketHashName   string           `json:"market_hash_name"`
	NotificationType NotificationType `json:"notification_type"`
	TargetPrice      *float64         `json:"target_price,omitempty"`
	OldPrice         float64          `json:"old_price"`
	CurrentPrice     float64          `json:"current_price"`
	PriceChange      float64          `json:"price_change"`
	SkinImageURL     string           `json:"skin_image_url"`
	Timestamp        time.Time        `json:"timestamp"`
}

func NewPriceAlertNotification(userID, skinID uuid.UUID, marketHashName string, oldPrice, currentPrice float64) *PriceAlertNotification {
	priceChange := 0.0
	if oldPrice > 0 {
		priceChange = ((currentPrice - oldPrice) / oldPrice) * 100
	}

	notifType := TypePriceDrop
	if currentPrice > oldPrice {
		notifType = TypePriceIncrease
	}

	return &PriceAlertNotification{
		UserID:           userID,
		SkinID:           skinID,
		MarketHashName:   marketHashName,
		NotificationType: notifType,
		OldPrice:         oldPrice,
		CurrentPrice:     currentPrice,
		PriceChange:      priceChange,
		Timestamp:        time.Now(),
	}
}

func (p *PriceAlertNotification) GenerateMessage() (title, message string) {
	switch p.NotificationType {
	case TypePriceDrop:
		title = "Price Drop Alert!"
		message = fmt.Sprintf(
			"%s dropped from $%.2f to $%.2f (%.1f%% down)",
			p.MarketHashName,
			p.OldPrice,
			p.CurrentPrice,
			-p.PriceChange,
		)
	case TypePriceIncrease:
		title = "Price Increase Alert!"
		message = fmt.Sprintf(
			"%s increased from $%.2f to $%.2f (%.1f%% up)",
			p.MarketHashName,
			p.OldPrice,
			p.CurrentPrice,
			p.PriceChange,
		)
	case TypeTargetReached:
		title = "Target Price Reached!"
		message = fmt.Sprintf(
			"%s reached your target price of $%.2f (current: $%.2f)",
			p.MarketHashName,
			*p.TargetPrice,
			p.CurrentPrice,
		)
	default:
		title = "Price Update"
		message = fmt.Sprintf("%s price updated to $%.2f", p.MarketHashName, p.CurrentPrice)
	}
	return
}

func (p *PriceAlertNotification) GetPriority() NotificationPriority {
	absChange := p.PriceChange
	if absChange < 0 {
		absChange = -absChange
	}

	switch {
	case absChange >= 20:
		return PriorityUrgent
	case absChange >= 10:
		return PriorityHigh
	case absChange >= 5:
		return PriorityNormal
	default:
		return PriorityLow
	}
}

type NotificationFilter struct {
	UserID   uuid.UUID
	Types    []NotificationType
	IsRead   *bool
	Priority *NotificationPriority
	Since    *time.Time
	Limit    int
	Offset   int
}

func NewNotificationFilter(userID uuid.UUID) *NotificationFilter {
	return &NotificationFilter{
		UserID: userID,
		Limit:  50,
		Offset: 0,
	}
}

type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	Total         int            `json:"total"`
	Unread        int            `json:"unread"`
	Page          int            `json:"page"`
	PageSize      int            `json:"page_size"`
}

type NotificationStats struct {
	Total      int                          `json:"total"`
	Unread     int                          `json:"unread"`
	ByType     map[NotificationType]int     `json:"by_type"`
	ByPriority map[NotificationPriority]int `json:"by_priority"`
	Last24h    int                          `json:"last_24h"`
}

type MarkReadRequest struct {
	NotificationIDs []uuid.UUID `json:"notification_ids" binding:"required"`
}

type NotificationPreferences struct {
	UserID            uuid.UUID            `json:"user_id" db:"user_id"`
	EnabledTypes      []NotificationType   `json:"enabled_types" db:"enabled_types"`
	MinPriceChange    float64              `json:"min_price_change" db:"min_price_change"`
	QuietHoursEnabled bool                 `json:"quiet_hours_enabled" db:"quiet_hours_enabled"`
	QuietHoursStart   string               `json:"quiet_hours_start" db:"quiet_hours_start"` // "22:00"
	QuietHoursEnd     string               `json:"quiet_hours_end" db:"quiet_hours_end"`     // "08:00"
	Channels          NotificationChannels `json:"channels" db:"channels"`
	UpdatedAt         time.Time            `json:"updated_at" db:"updated_at"`
}

type NotificationChannels struct {
	Email   bool `json:"email"`
	Push    bool `json:"push"`
	InApp   bool `json:"in_app"`
	Webhook bool `json:"webhook"`
}

func NewNotificationPreferences(userID uuid.UUID) *NotificationPreferences {
	return &NotificationPreferences{
		UserID: userID,
		EnabledTypes: []NotificationType{
			TypePriceDrop,
			TypeTargetReached,
		},
		MinPriceChange:    5.0,
		QuietHoursEnabled: false,
		Channels: NotificationChannels{
			Email:   true,
			Push:    false,
			InApp:   true,
			Webhook: false,
		},
		UpdatedAt: time.Now(),
	}
}

func (p *NotificationPreferences) IsInQuietHours() bool {
	if !p.QuietHoursEnabled {
		return false
	}

	now := time.Now()
	currentTime := now.Format("15:04")

	return currentTime >= p.QuietHoursStart && currentTime < p.QuietHoursEnd
}

func (p *NotificationPreferences) ShouldSend(notifType NotificationType, priceChange float64) bool {
	if p.IsInQuietHours() {
		return false
	}

	typeEnabled := false
	for _, t := range p.EnabledTypes {
		if t == notifType {
			typeEnabled = true
			break
		}
	}
	if !typeEnabled {
		return false
	}

	absChange := priceChange
	if absChange < 0 {
		absChange = -absChange
	}
	if absChange < p.MinPriceChange {
		return false
	}

	return true
}

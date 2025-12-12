package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/entity"
	"github.com/kedr891/cs-parser/pkg/logger"
	"github.com/kedr891/cs-parser/pkg/postgres"
)

// Repository - интерфейс репозитория для уведомлений
type Repository interface {
	CreateNotification(ctx context.Context, notification *entity.Notification) error
	GetNotifications(ctx context.Context, filter *entity.NotificationFilter) ([]entity.Notification, error)
	MarkNotificationsAsRead(ctx context.Context, userID uuid.UUID, notificationIDs []uuid.UUID) error
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	GetNotificationPreferences(ctx context.Context, userID uuid.UUID) (*entity.NotificationPreferences, error)
	UpdateNotificationPreferences(ctx context.Context, preferences *entity.NotificationPreferences) error
	DeleteOldNotifications(ctx context.Context, before time.Time) (int, error)
	GetNotificationStats(ctx context.Context, userID uuid.UUID) (*entity.NotificationStats, error)
}

// repository - реализация репозитория
type repository struct {
	pg  *postgres.Postgres
	log *logger.Logger
}

// NewRepository - создать репозиторий
func NewRepository(pg *postgres.Postgres, log *logger.Logger) Repository {
	return &repository{
		pg:  pg,
		log: log,
	}
}

// CreateNotification - создать уведомление
func (r *repository) CreateNotification(ctx context.Context, notification *entity.Notification) error {
	dataJSON, err := json.Marshal(notification.Data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}

	query := `
		INSERT INTO notifications (id, user_id, type, title, message, data, is_read, priority, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.pg.Pool.Exec(ctx, query,
		notification.ID,
		notification.UserID,
		notification.Type,
		notification.Title,
		notification.Message,
		dataJSON,
		notification.IsRead,
		notification.Priority,
		notification.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}

	return nil
}

// GetNotifications - получить уведомления с фильтром
func (r *repository) GetNotifications(ctx context.Context, filter *entity.NotificationFilter) ([]entity.Notification, error) {
	query := `
		SELECT id, user_id, type, title, message, data, is_read, priority, created_at, read_at
		FROM notifications
		WHERE user_id = $1
	`

	args := []interface{}{filter.UserID}
	argIndex := 2

	// Фильтр по прочитанности
	if filter.IsRead != nil {
		query += fmt.Sprintf(" AND is_read = $%d", argIndex)
		args = append(args, *filter.IsRead)
		argIndex++
	}

	// Фильтр по типам
	if len(filter.Types) > 0 {
		query += fmt.Sprintf(" AND type = ANY($%d)", argIndex)
		args = append(args, filter.Types)
		argIndex++
	}

	// Фильтр по приоритету
	if filter.Priority != nil {
		query += fmt.Sprintf(" AND priority = $%d", argIndex)
		args = append(args, *filter.Priority)
		argIndex++
	}

	// Фильтр по времени
	if filter.Since != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filter.Since)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pg.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query notifications: %w", err)
	}
	defer rows.Close()

	var notifications []entity.Notification
	for rows.Next() {
		var n entity.Notification
		var dataJSON []byte

		err := rows.Scan(
			&n.ID,
			&n.UserID,
			&n.Type,
			&n.Title,
			&n.Message,
			&dataJSON,
			&n.IsRead,
			&n.Priority,
			&n.CreatedAt,
			&n.ReadAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}

		// Десериализация data
		if len(dataJSON) > 0 {
			if err := json.Unmarshal(dataJSON, &n.Data); err != nil {
				r.log.Warn("Failed to unmarshal notification data", "error", err)
				n.Data = make(map[string]interface{})
			}
		} else {
			n.Data = make(map[string]interface{})
		}

		notifications = append(notifications, n)
	}

	return notifications, nil
}

// MarkNotificationsAsRead - пометить уведомления как прочитанные
func (r *repository) MarkNotificationsAsRead(ctx context.Context, userID uuid.UUID, notificationIDs []uuid.UUID) error {
	query := `
		UPDATE notifications
		SET is_read = true, read_at = NOW()
		WHERE user_id = $1 AND id = ANY($2)
	`

	_, err := r.pg.Pool.Exec(ctx, query, userID, notificationIDs)
	if err != nil {
		return fmt.Errorf("update notifications: %w", err)
	}

	return nil
}

// GetUnreadCount - получить количество непрочитанных
func (r *repository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`

	err := r.pg.Pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread: %w", err)
	}

	return count, nil
}

// GetNotificationPreferences - получить настройки уведомлений
func (r *repository) GetNotificationPreferences(ctx context.Context, userID uuid.UUID) (*entity.NotificationPreferences, error) {
	query := `
		SELECT 
			user_id, enabled_types, min_price_change, 
			quiet_hours_enabled, quiet_hours_start, quiet_hours_end,
			email_notifications, push_notifications, in_app_notifications, webhook_notifications,
			updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`

	var prefs entity.NotificationPreferences
	var enabledTypesJSON []byte

	err := r.pg.Pool.QueryRow(ctx, query, userID).Scan(
		&prefs.UserID,
		&enabledTypesJSON,
		&prefs.MinPriceChange,
		&prefs.QuietHoursEnabled,
		&prefs.QuietHoursStart,
		&prefs.QuietHoursEnd,
		&prefs.Channels.Email,
		&prefs.Channels.Push,
		&prefs.Channels.InApp,
		&prefs.Channels.Webhook,
		&prefs.UpdatedAt,
	)

	if err != nil {
		// Если настройки не найдены, вернуть дефолтные
		return entity.NewNotificationPreferences(userID), nil
	}

	// Десериализация типов
	if err := json.Unmarshal(enabledTypesJSON, &prefs.EnabledTypes); err != nil {
		r.log.Warn("Failed to unmarshal enabled types", "error", err)
		prefs.EnabledTypes = []entity.NotificationType{entity.TypePriceDrop, entity.TypeTargetReached}
	}

	return &prefs, nil
}

// UpdateNotificationPreferences - обновить настройки уведомлений
func (r *repository) UpdateNotificationPreferences(ctx context.Context, preferences *entity.NotificationPreferences) error {
	enabledTypesJSON, err := json.Marshal(preferences.EnabledTypes)
	if err != nil {
		return fmt.Errorf("marshal enabled types: %w", err)
	}

	query := `
		INSERT INTO notification_preferences (
			user_id, enabled_types, min_price_change,
			quiet_hours_enabled, quiet_hours_start, quiet_hours_end,
			email_notifications, push_notifications, in_app_notifications, webhook_notifications,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (user_id) DO UPDATE SET
			enabled_types = EXCLUDED.enabled_types,
			min_price_change = EXCLUDED.min_price_change,
			quiet_hours_enabled = EXCLUDED.quiet_hours_enabled,
			quiet_hours_start = EXCLUDED.quiet_hours_start,
			quiet_hours_end = EXCLUDED.quiet_hours_end,
			email_notifications = EXCLUDED.email_notifications,
			push_notifications = EXCLUDED.push_notifications,
			in_app_notifications = EXCLUDED.in_app_notifications,
			webhook_notifications = EXCLUDED.webhook_notifications,
			updated_at = NOW()
	`

	_, err = r.pg.Pool.Exec(ctx, query,
		preferences.UserID,
		enabledTypesJSON,
		preferences.MinPriceChange,
		preferences.QuietHoursEnabled,
		preferences.QuietHoursStart,
		preferences.QuietHoursEnd,
		preferences.Channels.Email,
		preferences.Channels.Push,
		preferences.Channels.InApp,
		preferences.Channels.Webhook,
		preferences.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("upsert preferences: %w", err)
	}

	return nil
}

// DeleteOldNotifications - удалить старые уведомления
func (r *repository) DeleteOldNotifications(ctx context.Context, before time.Time) (int, error) {
	query := `DELETE FROM notifications WHERE created_at < $1 AND is_read = true`

	result, err := r.pg.Pool.Exec(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("delete old notifications: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// GetNotificationStats - получить статистику уведомлений
func (r *repository) GetNotificationStats(ctx context.Context, userID uuid.UUID) (*entity.NotificationStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE is_read = false) as unread,
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '24 hours') as last_24h
		FROM notifications
		WHERE user_id = $1
	`

	var stats entity.NotificationStats
	err := r.pg.Pool.QueryRow(ctx, query, userID).Scan(
		&stats.Total,
		&stats.Unread,
		&stats.Last24h,
	)

	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	// Статистика по типам
	typeQuery := `
		SELECT type, COUNT(*) 
		FROM notifications 
		WHERE user_id = $1 
		GROUP BY type
	`

	rows, err := r.pg.Pool.Query(ctx, typeQuery, userID)
	if err != nil {
		return &stats, nil // Возвращаем базовую статистику
	}
	defer rows.Close()

	stats.ByType = make(map[entity.NotificationType]int)
	for rows.Next() {
		var notifType entity.NotificationType
		var count int
		if err := rows.Scan(&notifType, &count); err == nil {
			stats.ByType[notifType] = count
		}
	}

	// Статистика по приоритетам
	priorityQuery := `
		SELECT priority, COUNT(*) 
		FROM notifications 
		WHERE user_id = $1 
		GROUP BY priority
	`

	rows2, err := r.pg.Pool.Query(ctx, priorityQuery, userID)
	if err != nil {
		return &stats, nil
	}
	defer rows2.Close()

	stats.ByPriority = make(map[entity.NotificationPriority]int)
	for rows2.Next() {
		var priority entity.NotificationPriority
		var count int
		if err := rows2.Scan(&priority, &count); err == nil {
			stats.ByPriority[priority] = count
		}
	}

	return &stats, nil
}

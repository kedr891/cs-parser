package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/domain"
	"github.com/kedr891/cs-parser/internal/entity"
)

type Service struct {
	repo  Repository
	cache domain.CacheStorage
	log   domain.Logger
}

func NewService(
	repo Repository,
	cache domain.CacheStorage,
	log domain.Logger,
) *Service {
	return &Service{
		repo:  repo,
		cache: cache,
		log:   log,
	}
}

func (s *Service) CreateNotification(ctx context.Context, notification *entity.Notification) error {
	if err := s.repo.CreateNotification(ctx, notification); err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	s.log.Debug("Notification created",
		"id", notification.ID,
		"user_id", notification.UserID,
		"type", notification.Type,
	)

	return nil
}

func (s *Service) SendNotification(ctx context.Context, notification *entity.Notification, preferences *entity.NotificationPreferences) error {
	var errors []error

	if preferences.Channels.Email {
		if err := s.sendEmail(ctx, notification); err != nil {
			s.log.Error("Failed to send email", "error", err)
			errors = append(errors, err)
		}
	}

	if preferences.Channels.Push {
		if err := s.sendPush(ctx, notification); err != nil {
			s.log.Error("Failed to send push", "error", err)
			errors = append(errors, err)
		}
	}

	if preferences.Channels.Webhook {
		if err := s.sendWebhook(ctx, notification); err != nil {
			s.log.Error("Failed to send webhook", "error", err)
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send notification through %d channels", len(errors))
	}

	return nil
}

func (s *Service) sendEmail(ctx context.Context, notification *entity.Notification) error {
	// TODO: Реализовать интеграцию с email сервисом (SendGrid, AWS SES, etc.)
	s.log.Info("Email notification would be sent",
		"notification_id", notification.ID,
		"user_id", notification.UserID,
		"title", notification.Title,
	)
	return nil
}

func (s *Service) sendPush(ctx context.Context, notification *entity.Notification) error {
	// TODO: Реализовать интеграцию с push сервисом (Firebase, OneSignal, etc.)
	s.log.Info("Push notification would be sent",
		"notification_id", notification.ID,
		"user_id", notification.UserID,
		"title", notification.Title,
	)
	return nil
}

func (s *Service) sendWebhook(ctx context.Context, notification *entity.Notification) error {
	// TODO: Реализовать отправку webhook'а на URL пользователя
	s.log.Info("Webhook notification would be sent",
		"notification_id", notification.ID,
		"user_id", notification.UserID,
		"title", notification.Title,
	)
	return nil
}

func (s *Service) GetUserPreferences(ctx context.Context, userID uuid.UUID) (*entity.NotificationPreferences, error) {
	preferences, err := s.repo.GetNotificationPreferences(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get preferences: %w", err)
	}
	return preferences, nil
}

func (s *Service) WasAlertSent(ctx context.Context, userID, skinID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("alerts:sent:%s:%s", userID.String(), skinID.String())

	_, err := s.cache.Get(ctx, key)
	if err != nil {
		return false, nil
	}

	return true, nil
}

func (s *Service) TrackAlertSent(ctx context.Context, userID, skinID uuid.UUID) error {
	key := fmt.Sprintf("alerts:sent:%s:%s", userID.String(), skinID.String())

	return s.cache.Set(ctx, key, "1", time.Hour)
}

func (s *Service) GetNotifications(ctx context.Context, filter *entity.NotificationFilter) ([]entity.Notification, error) {
	notifications, err := s.repo.GetNotifications(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("get notifications: %w", err)
	}
	return notifications, nil
}

func (s *Service) MarkAsRead(ctx context.Context, userID uuid.UUID, notificationIDs []uuid.UUID) error {
	if err := s.repo.MarkNotificationsAsRead(ctx, userID, notificationIDs); err != nil {
		return fmt.Errorf("mark as read: %w", err)
	}
	return nil
}

func (s *Service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := s.repo.GetUnreadCount(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get unread count: %w", err)
	}
	return count, nil
}

func (s *Service) DeleteOldNotifications(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	deleted, err := s.repo.DeleteOldNotifications(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("delete old notifications: %w", err)
	}

	s.log.Info("Old notifications deleted", "count", deleted)
	return nil
}

func (s *Service) GetStats(ctx context.Context, userID uuid.UUID) (*entity.NotificationStats, error) {
	stats, err := s.repo.GetNotificationStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}
	return stats, nil
}

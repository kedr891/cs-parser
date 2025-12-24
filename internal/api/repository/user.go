package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kedr891/cs-parser/internal/entity"
	"github.com/kedr891/cs-parser/pkg/postgres"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *entity.User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	UserExistsByEmail(ctx context.Context, email string) (bool, error)
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error
	UpdateUser(ctx context.Context, user *entity.User) error

	CreateUserSettings(ctx context.Context, settings *entity.UserSettings) error
	GetUserSettings(ctx context.Context, userID uuid.UUID) (*entity.UserSettings, error)
	UpdateUserSettings(ctx context.Context, settings *entity.UserSettings) error

	GetWatchlistWithSkins(ctx context.Context, userID uuid.UUID) ([]entity.WatchlistWithSkin, error)
	WatchlistExists(ctx context.Context, userID, skinID uuid.UUID) (bool, error)
	CreateWatchlist(ctx context.Context, wl *entity.Watchlist) error
	DeleteWatchlist(ctx context.Context, userID, skinID uuid.UUID) error
	GetActiveWatchlists(ctx context.Context) ([]entity.Watchlist, error)

	GetNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit int) ([]entity.Notification, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	MarkNotificationsRead(ctx context.Context, userID uuid.UUID, notificationIDs []uuid.UUID) error

	GetUserStats(ctx context.Context, userID uuid.UUID) (*entity.UserStats, error)
}

type userRepository struct {
	pg *postgres.Postgres
}

func NewUserRepository(pg *postgres.Postgres) UserRepository {
	return &userRepository{
		pg: pg,
	}
}

func (r *userRepository) CreateUser(ctx context.Context, user *entity.User) error {
	query := `
		INSERT INTO users (id, email, username, password_hash, role, is_active, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pg.Pool.Exec(ctx, query,
		user.ID, user.Email, user.Username, user.PasswordHash,
		user.Role, user.IsActive, user.IsVerified,
		user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	query := `
		SELECT id, email, username, password_hash, role, is_active, is_verified, last_login_at, created_at, updated_at
		FROM users WHERE id = $1
	`
	var user entity.User
	err := r.pg.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Username, &user.PasswordHash,
		&user.Role, &user.IsActive, &user.IsVerified,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return &user, err
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT id, email, username, password_hash, role, is_active, is_verified, last_login_at, created_at, updated_at
		FROM users WHERE email = $1
	`
	var user entity.User
	err := r.pg.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Username, &user.PasswordHash,
		&user.Role, &user.IsActive, &user.IsVerified,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return &user, err
}

func (r *userRepository) UserExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.pg.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists)
	return exists, err
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pg.Pool.Exec(ctx, `UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`, userID)
	return err
}

func (r *userRepository) UpdateUser(ctx context.Context, user *entity.User) error {
	query := `
		UPDATE users 
		SET email = $1, username = $2, updated_at = $3
		WHERE id = $4
	`
	_, err := r.pg.Pool.Exec(ctx, query, user.Email, user.Username, user.UpdatedAt, user.ID)
	return err
}

func (r *userRepository) CreateUserSettings(ctx context.Context, settings *entity.UserSettings) error {
	query := `
		INSERT INTO user_settings (
			user_id, email_notifications, push_notifications,
			price_alert_threshold, preferred_currency, notification_frequency, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pg.Pool.Exec(ctx, query,
		settings.UserID, settings.EmailNotifications, settings.PushNotifications,
		settings.PriceAlertThreshold, settings.PreferredCurrency,
		settings.NotificationFrequency, settings.UpdatedAt,
	)
	return err
}

func (r *userRepository) GetUserSettings(ctx context.Context, userID uuid.UUID) (*entity.UserSettings, error) {
	query := `
		SELECT user_id, email_notifications, push_notifications,
			price_alert_threshold, preferred_currency, notification_frequency, updated_at
		FROM user_settings WHERE user_id = $1
	`
	var settings entity.UserSettings
	err := r.pg.Pool.QueryRow(ctx, query, userID).Scan(
		&settings.UserID, &settings.EmailNotifications, &settings.PushNotifications,
		&settings.PriceAlertThreshold, &settings.PreferredCurrency,
		&settings.NotificationFrequency, &settings.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("settings not found")
	}
	return &settings, err
}

func (r *userRepository) UpdateUserSettings(ctx context.Context, settings *entity.UserSettings) error {
	query := `
		UPDATE user_settings 
		SET email_notifications = $1, push_notifications = $2,
			price_alert_threshold = $3, preferred_currency = $4,
			notification_frequency = $5, updated_at = $6
		WHERE user_id = $7
	`
	_, err := r.pg.Pool.Exec(ctx, query,
		settings.EmailNotifications, settings.PushNotifications,
		settings.PriceAlertThreshold, settings.PreferredCurrency,
		settings.NotificationFrequency, settings.UpdatedAt, settings.UserID,
	)
	return err
}

func (r *userRepository) GetWatchlistWithSkins(ctx context.Context, userID uuid.UUID) ([]entity.WatchlistWithSkin, error) {
	query := `
		SELECT 
			w.id, w.user_id, w.skin_id, w.target_price, w.notify_on_drop, w.notify_on_price, w.is_active, w.added_at, w.updated_at,
			s.id, s.market_hash_name, s.name, s.weapon, s.quality, s.rarity, s.current_price, s.currency, s.image_url,
			s.volume_24h, s.price_change_24h, s.price_change_7d, s.lowest_price, s.highest_price,
			s.last_updated, s.created_at, s.updated_at
		FROM watchlist w
		JOIN skins s ON w.skin_id = s.id
		WHERE w.user_id = $1 AND w.is_active = true
		ORDER BY w.added_at DESC
	`

	rows, err := r.pg.Pool.Query(ctx, query, userID)
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
			&wws.Skin.Volume24h, &wws.Skin.PriceChange24h, &wws.Skin.PriceChange7d,
			&wws.Skin.LowestPrice, &wws.Skin.HighestPrice,
			&wws.Skin.LastUpdated, &wws.Skin.CreatedAt, &wws.Skin.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, wws)
	}

	return result, nil
}

func (r *userRepository) WatchlistExists(ctx context.Context, userID, skinID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pg.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM watchlist WHERE user_id = $1 AND skin_id = $2 AND is_active = true)`,
		userID, skinID,
	).Scan(&exists)
	return exists, err
}

func (r *userRepository) CreateWatchlist(ctx context.Context, wl *entity.Watchlist) error {
	query := `
		INSERT INTO watchlist (id, user_id, skin_id, target_price, notify_on_drop, notify_on_price, is_active, added_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pg.Pool.Exec(ctx, query,
		wl.ID, wl.UserID, wl.SkinID, wl.TargetPrice,
		wl.NotifyOnDrop, wl.NotifyOnPrice, wl.IsActive,
		wl.AddedAt, wl.UpdatedAt,
	)
	return err
}

func (r *userRepository) DeleteWatchlist(ctx context.Context, userID, skinID uuid.UUID) error {
	_, err := r.pg.Pool.Exec(ctx,
		`UPDATE watchlist SET is_active = false, updated_at = NOW() WHERE user_id = $1 AND skin_id = $2`,
		userID, skinID,
	)
	return err
}

func (r *userRepository) GetActiveWatchlists(ctx context.Context) ([]entity.Watchlist, error) {
	query := `
		SELECT id, user_id, skin_id, target_price, notify_on_drop, notify_on_price, is_active, added_at, updated_at
		FROM watchlist
		WHERE is_active = true
	`

	rows, err := r.pg.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var watchlists []entity.Watchlist
	for rows.Next() {
		var wl entity.Watchlist
		err := rows.Scan(
			&wl.ID, &wl.UserID, &wl.SkinID, &wl.TargetPrice,
			&wl.NotifyOnDrop, &wl.NotifyOnPrice, &wl.IsActive,
			&wl.AddedAt, &wl.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		watchlists = append(watchlists, wl)
	}

	return watchlists, nil
}

func (r *userRepository) GetNotifications(ctx context.Context, userID uuid.UUID, unreadOnly bool, limit int) ([]entity.Notification, error) {
	query := `
		SELECT id, user_id, type, title, message, is_read, priority, created_at, read_at
		FROM notifications
		WHERE user_id = $1
	`

	if unreadOnly {
		query += " AND is_read = false"
	}

	query += " ORDER BY created_at DESC LIMIT $2"

	rows, err := r.pg.Pool.Query(ctx, query, userID, limit)
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

func (r *userRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.pg.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`, userID).Scan(&count)
	return count, err
}

func (r *userRepository) MarkNotificationsRead(ctx context.Context, userID uuid.UUID, notificationIDs []uuid.UUID) error {
	query := `
		UPDATE notifications
		SET is_read = true, read_at = NOW()
		WHERE user_id = $1 AND id = ANY($2)
	`
	_, err := r.pg.Pool.Exec(ctx, query, userID, notificationIDs)
	return err
}

func (r *userRepository) GetUserStats(ctx context.Context, userID uuid.UUID) (*entity.UserStats, error) {
	query := `
		SELECT 
			(SELECT COUNT(*) FROM watchlist WHERE user_id = $1 AND is_active = true) as total_watchlist,
			(SELECT COUNT(*) FROM watchlist WHERE user_id = $1 AND is_active = true AND target_price IS NOT NULL) as active_alerts,
			(SELECT COUNT(*) FROM notifications WHERE user_id = $1) as total_notifications,
			(SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false) as unread_notifications,
			(SELECT created_at FROM users WHERE id = $1) as member_since
	`

	var stats entity.UserStats
	err := r.pg.Pool.QueryRow(ctx, query, userID).Scan(
		&stats.TotalWatchlist,
		&stats.ActiveAlerts,
		&stats.TotalNotifications,
		&stats.UnreadNotifications,
		&stats.MemberSince,
	)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

DROP TRIGGER IF EXISTS update_skins_updated_at ON skins;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_watchlist_updated_at ON watchlist;
DROP TRIGGER IF EXISTS update_user_settings_updated_at ON user_settings;
DROP TRIGGER IF EXISTS update_notification_preferences_updated_at ON notification_preferences;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_skins_weapon;
DROP INDEX IF EXISTS idx_skins_quality;
DROP INDEX IF EXISTS idx_skins_current_price;
DROP INDEX IF EXISTS idx_skins_volume_24h;
DROP INDEX IF EXISTS idx_skins_updated_at;
DROP INDEX IF EXISTS idx_skins_market_hash_name;

DROP INDEX IF EXISTS idx_price_history_skin_id;
DROP INDEX IF EXISTS idx_price_history_recorded_at;
DROP INDEX IF EXISTS idx_price_history_skin_recorded;

DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;

DROP INDEX IF EXISTS idx_watchlist_user_id;
DROP INDEX IF EXISTS idx_watchlist_skin_id;
DROP INDEX IF EXISTS idx_watchlist_active;

DROP INDEX IF EXISTS idx_notifications_user_id;
DROP INDEX IF EXISTS idx_notifications_created_at;
DROP INDEX IF EXISTS idx_notifications_is_read;
DROP INDEX IF EXISTS idx_notifications_user_unread;

DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS watchlist;
DROP TABLE IF EXISTS user_settings;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS price_history;
DROP TABLE IF EXISTS skins;
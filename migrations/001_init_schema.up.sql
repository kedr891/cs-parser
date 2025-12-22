-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Функция для генерации slug из market_hash_name
CREATE OR REPLACE FUNCTION generate_skin_slug(market_hash_name TEXT)
RETURNS TEXT AS $$
DECLARE
    slug TEXT;
    quality_abbr TEXT;
BEGIN
    -- Преобразовать в lowercase
    slug := LOWER(market_hash_name);
    
    -- Заменить качества на аббревиатуры
    slug := REPLACE(slug, '(factory new)', 'fn');
    slug := REPLACE(slug, '(minimal wear)', 'mw');
    slug := REPLACE(slug, '(field-tested)', 'ft');
    slug := REPLACE(slug, '(well-worn)', 'ww');
    slug := REPLACE(slug, '(battle-scarred)', 'bs');
    
    -- Убрать скобки и спецсимволы
    slug := REGEXP_REPLACE(slug, '[|()\[\]{}]', '', 'g');
    
    -- Заменить пробелы, дефисы и другие спецсимволы на подчеркивание
    slug := REGEXP_REPLACE(slug, '[^a-z0-9]+', '_', 'g');
    
    -- Убрать лишние подчеркивания в начале и конце
    slug := TRIM(BOTH '_' FROM slug);
    
    -- Убрать множественные подчеркивания
    slug := REGEXP_REPLACE(slug, '_+', '_', 'g');
    
    RETURN slug;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Trigger функция для автоматической генерации slug
CREATE OR REPLACE FUNCTION set_skin_slug()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.slug IS NULL OR NEW.slug = '' THEN
        NEW.slug := generate_skin_slug(NEW.market_hash_name);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger функция для обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

-- Skins table
CREATE TABLE IF NOT EXISTS skins (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug VARCHAR(255) UNIQUE NOT NULL,
    market_hash_name VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    weapon VARCHAR(100) NOT NULL,
    quality VARCHAR(50) NOT NULL,
    rarity VARCHAR(50),
    current_price DECIMAL(10,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    image_url TEXT,
    volume_24h INT NOT NULL DEFAULT 0,
    price_change_24h DECIMAL(10,2) NOT NULL DEFAULT 0,
    price_change_7d DECIMAL(10,2) NOT NULL DEFAULT 0,
    lowest_price DECIMAL(10,2) NOT NULL DEFAULT 0,
    highest_price DECIMAL(10,2) NOT NULL DEFAULT 0,
    last_updated TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Price history table
CREATE TABLE IF NOT EXISTS price_history (
    id BIGSERIAL PRIMARY KEY,
    skin_id UUID NOT NULL REFERENCES skins(id) ON DELETE CASCADE,
    price DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    source VARCHAR(50) NOT NULL,
    volume INT NOT NULL DEFAULT 0,
    recorded_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- User settings table
CREATE TABLE IF NOT EXISTS user_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    email_notifications BOOLEAN NOT NULL DEFAULT true,
    push_notifications BOOLEAN NOT NULL DEFAULT false,
    price_alert_threshold DECIMAL(5,2) NOT NULL DEFAULT 5.0,
    preferred_currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    notification_frequency VARCHAR(20) NOT NULL DEFAULT 'instant',
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Watchlist table
CREATE TABLE IF NOT EXISTS watchlist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skin_id UUID NOT NULL REFERENCES skins(id) ON DELETE CASCADE,
    target_price DECIMAL(10,2),
    notify_on_drop BOOLEAN NOT NULL DEFAULT true,
    notify_on_price BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    added_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, skin_id)
);

-- Notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    data JSONB,
    is_read BOOLEAN NOT NULL DEFAULT false,
    priority VARCHAR(20) NOT NULL DEFAULT 'normal',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    read_at TIMESTAMP
);

-- Notification preferences table
CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    enabled_types JSONB NOT NULL DEFAULT '["price_drop", "target_reached"]'::jsonb,
    min_price_change DECIMAL(5,2) NOT NULL DEFAULT 5.0,
    quiet_hours_enabled BOOLEAN NOT NULL DEFAULT false,
    quiet_hours_start VARCHAR(5) DEFAULT '22:00',
    quiet_hours_end VARCHAR(5) DEFAULT '08:00',
    email_notifications BOOLEAN NOT NULL DEFAULT true,
    push_notifications BOOLEAN NOT NULL DEFAULT false,
    in_app_notifications BOOLEAN NOT NULL DEFAULT true,
    webhook_notifications BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Skins indexes
CREATE INDEX IF NOT EXISTS idx_skins_slug ON skins(slug);              -- ✅ НОВЫЙ ИНДЕКС
CREATE INDEX IF NOT EXISTS idx_skins_weapon ON skins(weapon);
CREATE INDEX IF NOT EXISTS idx_skins_quality ON skins(quality);
CREATE INDEX IF NOT EXISTS idx_skins_current_price ON skins(current_price);
CREATE INDEX IF NOT EXISTS idx_skins_volume_24h ON skins(volume_24h);
CREATE INDEX IF NOT EXISTS idx_skins_updated_at ON skins(updated_at);
CREATE INDEX IF NOT EXISTS idx_skins_market_hash_name ON skins(market_hash_name);

-- Price history indexes
CREATE INDEX IF NOT EXISTS idx_price_history_skin_id ON price_history(skin_id);
CREATE INDEX IF NOT EXISTS idx_price_history_recorded_at ON price_history(recorded_at);
CREATE INDEX IF NOT EXISTS idx_price_history_skin_recorded ON price_history(skin_id, recorded_at);

-- Users indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

-- Watchlist indexes
CREATE INDEX IF NOT EXISTS idx_watchlist_user_id ON watchlist(user_id);
CREATE INDEX IF NOT EXISTS idx_watchlist_skin_id ON watchlist(skin_id);
CREATE INDEX IF NOT EXISTS idx_watchlist_active ON watchlist(is_active) WHERE is_active = true;

-- Notifications indexes
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);
CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications(is_read) WHERE is_read = false;
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread ON notifications(user_id, is_read) WHERE is_read = false;

-- Auto-generate slug for skins
CREATE TRIGGER trigger_set_skin_slug
BEFORE INSERT OR UPDATE ON skins
FOR EACH ROW
EXECUTE FUNCTION set_skin_slug();

-- Update updated_at timestamps
CREATE TRIGGER update_skins_updated_at 
BEFORE UPDATE ON skins
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at 
BEFORE UPDATE ON users
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_watchlist_updated_at 
BEFORE UPDATE ON watchlist
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_settings_updated_at 
BEFORE UPDATE ON user_settings
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notification_preferences_updated_at 
BEFORE UPDATE ON notification_preferences
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE skins IS 'CS2 skins with prices and statistics';
COMMENT ON COLUMN skins.slug IS 'Human-readable URL slug (e.g., awp_acheron_ft)';
COMMENT ON COLUMN skins.market_hash_name IS 'Steam Market unique identifier';

COMMENT ON TABLE price_history IS 'Historical price data for skins';
COMMENT ON TABLE users IS 'Application users';
COMMENT ON TABLE watchlist IS 'User watchlist for price alerts';
COMMENT ON TABLE notifications IS 'User notifications';
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE OR REPLACE FUNCTION generate_skin_slug(market_hash_name TEXT)
RETURNS TEXT AS $$
DECLARE
    slug TEXT;
    quality_abbr TEXT;
BEGIN
    slug := LOWER(market_hash_name);

    slug := REPLACE(slug, '(factory new)', 'fn');
    slug := REPLACE(slug, '(minimal wear)', 'mw');
    slug := REPLACE(slug, '(field-tested)', 'ft');
    slug := REPLACE(slug, '(well-worn)', 'ww');
    slug := REPLACE(slug, '(battle-scarred)', 'bs');

    slug := REGEXP_REPLACE(slug, '[|()\[\]{}]', '', 'g');

    slug := REGEXP_REPLACE(slug, '[^a-z0-9]+', '_', 'g');

    slug := TRIM(BOTH '_' FROM slug);
    
    slug := REGEXP_REPLACE(slug, '_+', '_', 'g');
    
    RETURN slug;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION set_skin_slug()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.slug IS NULL OR NEW.slug = '' THEN
        NEW.slug := generate_skin_slug(NEW.market_hash_name);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';

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

CREATE TABLE IF NOT EXISTS price_history (
    id BIGSERIAL PRIMARY KEY,
    skin_id UUID NOT NULL REFERENCES skins(id) ON DELETE CASCADE,
    price DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    source VARCHAR(50) NOT NULL,
    volume INT NOT NULL DEFAULT 0,
    recorded_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_skins_slug ON skins(slug);
CREATE INDEX IF NOT EXISTS idx_skins_weapon ON skins(weapon);
CREATE INDEX IF NOT EXISTS idx_skins_quality ON skins(quality);
CREATE INDEX IF NOT EXISTS idx_skins_current_price ON skins(current_price);
CREATE INDEX IF NOT EXISTS idx_skins_volume_24h ON skins(volume_24h);
CREATE INDEX IF NOT EXISTS idx_skins_updated_at ON skins(updated_at);
CREATE INDEX IF NOT EXISTS idx_skins_market_hash_name ON skins(market_hash_name);

CREATE INDEX IF NOT EXISTS idx_price_history_skin_id ON price_history(skin_id);
CREATE INDEX IF NOT EXISTS idx_price_history_recorded_at ON price_history(recorded_at);
CREATE INDEX IF NOT EXISTS idx_price_history_skin_recorded ON price_history(skin_id, recorded_at);

DROP TRIGGER IF EXISTS trigger_set_skin_slug ON skins;
CREATE TRIGGER trigger_set_skin_slug
BEFORE INSERT OR UPDATE ON skins
FOR EACH ROW
EXECUTE FUNCTION set_skin_slug();

DROP TRIGGER IF EXISTS update_skins_updated_at ON skins;
CREATE TRIGGER update_skins_updated_at 
BEFORE UPDATE ON skins
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE skins IS 'CS2 skins with prices and statistics';
COMMENT ON COLUMN skins.slug IS 'Human-readable URL slug (e.g., awp_acheron_ft)';
COMMENT ON COLUMN skins.market_hash_name IS 'Steam Market unique identifier';

COMMENT ON TABLE price_history IS 'Historical price data for skins';


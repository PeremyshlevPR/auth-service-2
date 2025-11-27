-- Drop triggers first
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop tables in correct order (respecting foreign keys)
-- First, drop dependent tables
DROP TABLE IF EXISTS oauth_providers CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
-- Then, drop the main table
DROP TABLE IF EXISTS users CASCADE;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;


-- Drop indexes
DROP INDEX IF EXISTS idx_oauth_providers_provider;
DROP INDEX IF EXISTS idx_oauth_providers_user_id;

-- Drop table
DROP TABLE IF EXISTS oauth_providers;


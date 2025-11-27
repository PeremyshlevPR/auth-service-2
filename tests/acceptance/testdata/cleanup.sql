-- Clean tables in correct order (respecting foreign keys)
-- First, truncate dependent tables
TRUNCATE TABLE oauth_providers CASCADE;
TRUNCATE TABLE refresh_tokens CASCADE;
-- Then, truncate the main table
TRUNCATE TABLE users CASCADE;


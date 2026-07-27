-- First admin user setup script
-- Run this after Docker is up:
--   docker compose exec db psql -U postgres -d aiswitch -f /tmp/init.sql
-- Or connect directly:
--   psql -h localhost -U postgres -d aiswitch

-- Create admin user (password: admin123, hash is bcrypt of 'admin123')
-- Change the email and password below!

-- To generate a bcrypt hash for your own password, run in Go:
--   go run -mod=mod golang.org/x/crypto/bcrypt
-- Or use any online bcrypt generator with cost 10.

-- Example bcrypt hash for password "admin123":
INSERT INTO users (email, password_hash, nickname, is_admin)
VALUES ('admin@aiswitch.app', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Admin', true)
ON CONFLICT (email) DO NOTHING;

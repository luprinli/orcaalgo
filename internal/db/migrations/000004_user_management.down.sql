-- DOWN: Remove user management tables
DROP TABLE IF EXISTS notification_settings;
DROP TABLE IF EXISTS email_verification_tokens;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS users;

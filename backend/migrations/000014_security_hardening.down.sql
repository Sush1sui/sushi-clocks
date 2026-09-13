ALTER TABLE users DROP CONSTRAINT IF EXISTS uq_users_email;
ALTER TABLE users ADD CONSTRAINT uq_users_company_email UNIQUE (company_id, email);
ALTER TABLE users DROP COLUMN IF EXISTS token_version;

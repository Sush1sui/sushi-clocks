ALTER TABLE users DROP CONSTRAINT IF EXISTS users_system_role_check;
ALTER TABLE users ADD CONSTRAINT users_system_role_check CHECK (system_role IN ('admin', 'hr', 'employee'));

-- +goose Up
ALTER TABLE app_roles ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE app_roles SET is_system = TRUE WHERE name IN ('user', 'company_store_manager', 'super_admin');

ALTER TABLE users ADD COLUMN role_id UUID REFERENCES app_roles(id);
UPDATE users u SET role_id = ar.id FROM app_roles ar WHERE ar.name = u.app_role;
ALTER TABLE users ALTER COLUMN role_id SET NOT NULL;

CREATE INDEX users_role_idx ON users (role_id);

-- +goose Down
DROP INDEX users_role_idx;
ALTER TABLE users DROP COLUMN role_id;
ALTER TABLE app_roles DROP COLUMN is_system;

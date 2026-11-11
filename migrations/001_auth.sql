-- migrate:up
CREATE TABLE app_roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT UNIQUE NOT NULL,
    description TEXT NOT NULL
);

CREATE TABLE role_policies (
    role_id  UUID NOT NULL REFERENCES app_roles(id) ON DELETE CASCADE,
    resource TEXT NOT NULL,
    action   TEXT NOT NULL,
    PRIMARY KEY (role_id, resource, action)
);

CREATE TABLE users (
    id            UUID PRIMARY KEY,
    name          TEXT NOT NULL,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role_id       UUID NOT NULL REFERENCES app_roles(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX users_role_idx ON users (role_id);

INSERT INTO app_roles (name, description) VALUES
    ('user',                  'Only the projects they are a member of'),
    ('company_store_manager', 'User permissions plus the company store'),
    ('super_admin',           'Every project and the company store');

-- Seed app-scope policies (read is implied by write in the checker).
INSERT INTO role_policies (role_id, resource, action)
SELECT r.id, p.resource, 'write'
FROM app_roles r
JOIN (VALUES
    ('user',                  'materials'),
    ('user',                  'projects'),
    ('user',                  'tasks'),
    ('user',                  'project_store'),
    ('user',                  'project_deliveries'),
    ('company_store_manager', 'materials'),
    ('company_store_manager', 'projects'),
    ('company_store_manager', 'tasks'),
    ('company_store_manager', 'project_store'),
    ('company_store_manager', 'project_deliveries'),
    ('company_store_manager', 'company_store'),
    ('company_store_manager', 'company_store_deliveries'),
    ('super_admin',           'materials'),
    ('super_admin',           'projects'),
    ('super_admin',           'tasks'),
    ('super_admin',           'project_store'),
    ('super_admin',           'project_deliveries'),
    ('super_admin',           'company_store'),
    ('super_admin',           'company_store_deliveries'),
    ('super_admin',           'roles'),
    ('super_admin',           'users')
) AS p(role, resource) ON p.role = r.name;

-- migrate:down
DROP TABLE users;
DROP TABLE role_policies;
DROP TABLE app_roles;

-- migrate:up
CREATE TABLE app_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    description TEXT NOT NULL
);

CREATE TABLE role_policies (
    role_id UUID REFERENCES app_roles(id) ON DELETE CASCADE,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    PRIMARY KEY (role_id, resource, action)
);

INSERT INTO app_roles (id, name, description) VALUES
(gen_random_uuid(), 'user', 'Can only access projects they are a member of'),
(gen_random_uuid(), 'company_store_manager', 'User permissions plus full access to company store'),
(gen_random_uuid(), 'super_admin', 'Full access to all projects and company store');

INSERT INTO role_policies (role_id, resource, action)
SELECT id, 'company_store', 'read' FROM app_roles WHERE name IN ('company_store_manager', 'super_admin');

INSERT INTO role_policies (role_id, resource, action)
SELECT id, 'company_store', 'write' FROM app_roles WHERE name IN ('company_store_manager', 'super_admin');

INSERT INTO role_policies (role_id, resource, action)
SELECT id, 'all_projects', 'read' FROM app_roles WHERE name = 'super_admin';

-- migrate:down
DROP TABLE role_policies;
DROP TABLE app_roles;

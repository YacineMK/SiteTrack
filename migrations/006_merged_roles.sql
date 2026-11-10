-- migrate:up

ALTER TABLE app_roles RENAME TO roles;
ALTER TABLE roles
    ADD COLUMN project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    ADD COLUMN is_admin   BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN is_system  BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE roles SET is_system = TRUE
WHERE name IN ('user', 'company_store_manager', 'super_admin');

ALTER TABLE roles DROP CONSTRAINT app_roles_name_key;
CREATE UNIQUE INDEX roles_scope_name_idx ON roles (project_id, name) NULLS NOT DISTINCT;

INSERT INTO roles (id, name, description, project_id, is_admin, is_system)
SELECT gen_random_uuid(), 'Admin', 'Project admin', id, TRUE, TRUE FROM projects
UNION ALL
SELECT gen_random_uuid(), 'Member', 'Project member', id, FALSE, TRUE FROM projects;

INSERT INTO role_policies (role_id, resource, action)
SELECT id, 'tasks', 'write'       FROM roles WHERE is_system AND name = 'Member'
UNION ALL
SELECT id, 'store', 'write'       FROM roles WHERE is_system AND name = 'Member'
UNION ALL
SELECT id, 'deliveries', 'write'  FROM roles WHERE is_system AND name = 'Member';

ALTER TABLE project_members ADD COLUMN role_id UUID REFERENCES roles(id);

UPDATE project_members pm SET role_id = r.id
FROM roles r
WHERE r.project_id = pm.project_id
  AND r.is_system
  AND r.name = CASE WHEN pm.is_admin THEN 'Admin' ELSE 'Member' END;

ALTER TABLE project_members
    ALTER COLUMN role_id SET NOT NULL,
    DROP COLUMN is_admin;

-- migrate:down
ALTER TABLE project_members ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE project_members pm SET is_admin = r.is_admin FROM roles r WHERE r.id = pm.role_id;
ALTER TABLE project_members DROP COLUMN role_id;
DELETE FROM roles WHERE project_id IS NOT NULL;
DROP INDEX roles_scope_name_idx;
CREATE UNIQUE INDEX app_roles_name_key ON roles (name);
ALTER TABLE roles
    DROP COLUMN updated_at,
    DROP COLUMN is_system,
    DROP COLUMN is_admin,
    DROP COLUMN project_id;
ALTER TABLE roles RENAME TO app_roles;

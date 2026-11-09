-- +goose Up
INSERT INTO role_policies (role_id, resource, action)
SELECT id, 'all_projects', 'write' FROM app_roles WHERE name = 'super_admin'
ON CONFLICT DO NOTHING;

DELETE FROM role_policies rp
USING app_roles ar
WHERE rp.role_id = ar.id
  AND ar.name IN ('super_admin', 'company_store_manager')
  AND rp.action = 'read'
  AND rp.resource IN ('all_projects', 'company_store');

-- +goose Down
INSERT INTO role_policies (role_id, resource, action)
SELECT id, 'all_projects', 'read' FROM app_roles WHERE name = 'super_admin'
ON CONFLICT DO NOTHING;

INSERT INTO role_policies (role_id, resource, action)
SELECT id, 'company_store', 'read' FROM app_roles WHERE name IN ('company_store_manager', 'super_admin')
ON CONFLICT DO NOTHING;

DELETE FROM role_policies rp
USING app_roles ar
WHERE rp.role_id = ar.id
  AND ar.name = 'super_admin'
  AND rp.resource = 'all_projects'
  AND rp.action = 'write';

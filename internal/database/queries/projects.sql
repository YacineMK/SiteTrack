-- name: CreateProject :one
INSERT INTO projects (id, display_id, name, description, planned_start_date, planned_end_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at;

-- name: GetProjectByDisplayID :one
SELECT id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at,
       member_count, task_count_not_started, task_count_in_progress, task_count_done
FROM projects
WHERE display_id = $1;

-- name: UpdateProject :one
UPDATE projects
SET name = $2, description = $3, planned_start_date = $4, planned_end_date = $5, updated_at = NOW()
WHERE display_id = $1
RETURNING id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at,
          member_count, task_count_not_started, task_count_in_progress, task_count_done;

-- name: DeleteProject :execrows
DELETE FROM projects WHERE display_id = $1;

-- Bounded at 2k projects (brief ceiling). OFFSET keeps the query small; late/progress
-- are computed per row from the stored task counts + a partial-index subquery.
-- name: ListProjects :many
WITH visible AS (
    SELECT p.id, p.display_id, p.name, p.description,
           p.planned_start_date, p.planned_end_date, p.created_at, p.updated_at,
           p.member_count, p.task_count_not_started, p.task_count_in_progress, p.task_count_done,
           (SELECT COUNT(*) FROM tasks t WHERE t.project_id = p.id
              AND ((t.status='not_started' AND t.planned_start_date < CURRENT_DATE)
                OR (t.status='in_progress' AND t.planned_end_date   < CURRENT_DATE)))::int AS late_count,
           (SELECT COUNT(*) FROM tasks t WHERE t.project_id = p.id AND t.blocked AND t.status <> 'done')::int AS blocked_count,
           CASE WHEN (p.task_count_not_started + p.task_count_in_progress + p.task_count_done) = 0 THEN 0
                ELSE round(p.task_count_done * 100.0
                     / (p.task_count_not_started + p.task_count_in_progress + p.task_count_done), 2) END::float8 AS progress,
           EXISTS (SELECT 1 FROM project_members pm
                   JOIN roles r ON r.id = pm.role_id
                   WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id) AND r.is_admin) AS is_admin
    FROM projects p
    WHERE (sqlc.arg(super_admin)::bool
           OR EXISTS (SELECT 1 FROM project_members pm
                      WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)))
      AND (sqlc.arg(q)::text = ''
           OR p.name       ILIKE '%' || sqlc.arg(q) || '%'
           OR p.display_id ILIKE '%' || sqlc.arg(q) || '%')
)
SELECT * FROM visible
WHERE NOT sqlc.arg(late_only)::bool OR late_count > 0
ORDER BY
    CASE sqlc.arg(sort)::text WHEN 'name'       THEN name                END ASC,
    CASE sqlc.arg(sort)::text WHEN 'start_date' THEN planned_start_date  END ASC,
    CASE sqlc.arg(sort)::text WHEN 'progress'   THEN progress            END DESC,
    CASE sqlc.arg(sort)::text WHEN 'late'       THEN late_count          END DESC,
    created_at DESC, id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountProjects :one
SELECT COUNT(*) FROM projects p
WHERE (sqlc.arg(super_admin)::bool
       OR EXISTS (SELECT 1 FROM project_members pm
                  WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)))
  AND (sqlc.arg(q)::text = ''
       OR p.name       ILIKE '%' || sqlc.arg(q) || '%'
       OR p.display_id ILIKE '%' || sqlc.arg(q) || '%')
  AND (NOT sqlc.arg(late_only)::bool
       OR EXISTS (SELECT 1 FROM tasks t WHERE t.project_id = p.id
            AND ((t.status='not_started' AND t.planned_start_date < CURRENT_DATE)
              OR (t.status='in_progress' AND t.planned_end_date   < CURRENT_DATE))));

-- Assign by role_id (admin or member role selected by caller).
-- name: AddProjectMember :exec
INSERT INTO project_members (project_id, user_id, role_id) VALUES ($1, $2, $3);

-- name: AddProjectMemberIfAbsent :exec
INSERT INTO project_members (project_id, user_id, role_id) VALUES ($1, $2, $3)
ON CONFLICT (project_id, user_id) DO NOTHING;

-- name: GetProjectMembership :one
SELECT r.is_admin FROM project_members pm
JOIN roles r ON r.id = pm.role_id
WHERE pm.project_id = $1 AND pm.user_id = $2;

-- name: ListProjectMembers :many
SELECT pm.user_id, u.name, u.email, r.id AS role_id, r.name AS role_name, r.is_admin, pm.created_at
FROM project_members pm
JOIN users u ON u.id = pm.user_id
JOIN roles r ON r.id = pm.role_id
WHERE pm.project_id = $1
ORDER BY pm.created_at, u.name;

-- name: SetProjectMemberRole :execrows
UPDATE project_members SET role_id = $3 WHERE project_id = $1 AND user_id = $2;

-- name: RemoveProjectMember :execrows
DELETE FROM project_members WHERE project_id = $1 AND user_id = $2;

-- name: LockProjectAdmins :many
SELECT pm.user_id FROM project_members pm
JOIN roles r ON r.id = pm.role_id
WHERE pm.project_id = $1 AND r.is_admin
FOR UPDATE;

-- name: GetProjectSystemRole :one
SELECT id FROM roles WHERE project_id = sqlc.arg(project_id)::uuid AND is_system AND name = sqlc.arg(name);

-- name: SeedProjectSystemRoles :exec
WITH new_roles AS (
    INSERT INTO roles (id, name, description, project_id, is_admin, is_system)
    VALUES (gen_random_uuid(), 'Admin',  'Project admin',  sqlc.arg(project_id)::uuid, TRUE,  TRUE),
           (gen_random_uuid(), 'Member', 'Project member', sqlc.arg(project_id)::uuid, FALSE, TRUE)
    RETURNING id, name
)
INSERT INTO role_policies (role_id, resource, action)
SELECT n.id, res, 'write'
FROM new_roles n
CROSS JOIN (VALUES ('tasks'),('store'),('deliveries')) AS t(res)
WHERE n.name = 'Member';

-- name: FindUserByEmail :one
SELECT id, name, email FROM users WHERE email = $1;

-- name: GetProjectMemberPolicies :many
SELECT rp.resource, rp.action, r.is_admin
FROM project_members pm
JOIN roles r ON r.id = pm.role_id
LEFT JOIN role_policies rp ON rp.role_id = r.id
WHERE pm.project_id = $1 AND pm.user_id = $2;

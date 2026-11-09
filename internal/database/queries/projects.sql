-- name: CreateProject :one
INSERT INTO projects (id, display_id, name, description, planned_start_date, planned_end_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at, member_count, task_count_not_started, task_count_in_progress, task_count_done, late_count, blocked_count, progress;

-- name: GetProjectByDisplayID :one
SELECT id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at, member_count, task_count_not_started, task_count_in_progress, task_count_done, late_count, blocked_count, progress
FROM projects
WHERE display_id = $1;

-- name: ListProjects :many
WITH cur AS (
    SELECT id, name, planned_start_date, progress, late_count, created_at
    FROM projects
    WHERE display_id = sqlc.arg(after_id)
)
SELECT p.id, p.display_id, p.name, p.description, p.planned_start_date, p.planned_end_date,
       p.created_at, p.updated_at,
       EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)) AS is_admin,
       p.member_count,
       p.task_count_not_started AS not_started,
       p.task_count_in_progress AS in_progress,
       p.task_count_done AS done,
       p.late_count AS late,
       p.blocked_count,
       p.progress
FROM projects p
LEFT JOIN cur ON TRUE
WHERE (sqlc.arg(super_admin)::boolean
   OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)))
  AND (NOT sqlc.arg(late_only)::boolean OR p.late_count > 0)
  AND (sqlc.arg(after_id)::text = '' OR (cur.id IS NOT NULL AND CASE
        WHEN sqlc.arg(sort_by)::text = 'name' AND sqlc.arg(sort_dir)::text = 'desc' THEN (p.name, p.id) < (cur.name, cur.id)
        WHEN sqlc.arg(sort_by)::text = 'name' THEN (p.name, p.id) > (cur.name, cur.id)
        WHEN sqlc.arg(sort_by)::text = 'start_date' AND sqlc.arg(sort_dir)::text = 'desc' THEN (p.planned_start_date, p.id) < (cur.planned_start_date, cur.id)
        WHEN sqlc.arg(sort_by)::text = 'start_date' THEN (p.planned_start_date, p.id) > (cur.planned_start_date, cur.id)
        WHEN sqlc.arg(sort_by)::text = 'progress' AND sqlc.arg(sort_dir)::text = 'desc' THEN (p.progress, p.id) < (cur.progress, cur.id)
        WHEN sqlc.arg(sort_by)::text = 'progress' THEN (p.progress, p.id) > (cur.progress, cur.id)
        WHEN sqlc.arg(sort_by)::text = 'late' AND sqlc.arg(sort_dir)::text = 'desc' THEN (p.late_count, p.id) < (cur.late_count, cur.id)
        WHEN sqlc.arg(sort_by)::text = 'late' THEN (p.late_count, p.id) > (cur.late_count, cur.id)
        WHEN sqlc.arg(sort_by)::text = 'created' AND sqlc.arg(sort_dir)::text = 'desc' THEN (p.created_at, p.id) < (cur.created_at, cur.id)
        WHEN sqlc.arg(sort_by)::text = 'created' THEN (p.created_at, p.id) > (cur.created_at, cur.id)
        ELSE p.id > cur.id
      END))
ORDER BY
  CASE WHEN sqlc.arg(sort_by)::text = 'created' AND sqlc.arg(sort_dir)::text = 'asc' THEN p.created_at END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'created' AND sqlc.arg(sort_dir)::text = 'desc' THEN p.created_at END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'name' AND sqlc.arg(sort_dir)::text = 'asc' THEN p.name END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'name' AND sqlc.arg(sort_dir)::text = 'desc' THEN p.name END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'start_date' AND sqlc.arg(sort_dir)::text = 'asc' THEN p.planned_start_date END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'start_date' AND sqlc.arg(sort_dir)::text = 'desc' THEN p.planned_start_date END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'progress' AND sqlc.arg(sort_dir)::text = 'asc' THEN p.progress END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'progress' AND sqlc.arg(sort_dir)::text = 'desc' THEN p.progress END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'late' AND sqlc.arg(sort_dir)::text = 'asc' THEN p.late_count END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'late' AND sqlc.arg(sort_dir)::text = 'desc' THEN p.late_count END DESC,
  CASE WHEN sqlc.arg(sort_dir)::text = 'desc' THEN p.id END DESC,
  CASE WHEN sqlc.arg(sort_dir)::text = 'asc' THEN p.id END ASC
LIMIT sqlc.arg(page_limit);

-- name: CountProjects :one
SELECT COUNT(*)
FROM projects p
WHERE (sqlc.arg(super_admin)::boolean
   OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)))
  AND (NOT sqlc.arg(late_only)::boolean OR p.late_count > 0);

-- name: SearchProjects :many
WITH cur AS (
    SELECT id, ts_rank_cd(search, websearch_to_tsquery('english', sqlc.arg(query))) AS rank
    FROM projects
    WHERE display_id = sqlc.arg(after_id)
)
SELECT p.id, p.display_id, p.name, p.description, p.planned_start_date, p.planned_end_date,
       p.created_at, p.updated_at,
       ts_rank_cd(p.search, websearch_to_tsquery('english', sqlc.arg(query))) AS rank,
       EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)) AS is_admin,
       p.member_count,
       p.task_count_not_started AS not_started,
       p.task_count_in_progress AS in_progress,
       p.task_count_done AS done,
       p.late_count AS late,
       p.blocked_count,
       p.progress
FROM projects p
LEFT JOIN cur ON TRUE
WHERE (sqlc.arg(super_admin)::boolean
   OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)))
  AND (p.search @@ websearch_to_tsquery('english', sqlc.arg(query))
   OR p.name ILIKE '%' || sqlc.arg(query) || '%'
   OR p.display_id ILIKE '%' || sqlc.arg(query) || '%')
  AND (NOT sqlc.arg(late_only)::boolean OR p.late_count > 0)
  AND (sqlc.arg(after_id)::text = ''
   OR (cur.id IS NOT NULL AND (ts_rank_cd(p.search, websearch_to_tsquery('english', sqlc.arg(query))), p.id) < (cur.rank, cur.id)))
ORDER BY rank DESC, p.id DESC
LIMIT sqlc.arg(page_limit);

-- name: CountSearchProjects :one
SELECT COUNT(*)
FROM projects p
WHERE (sqlc.arg(super_admin)::boolean
   OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)))
  AND (p.search @@ websearch_to_tsquery('english', sqlc.arg(query))
   OR p.name ILIKE '%' || sqlc.arg(query) || '%'
   OR p.display_id ILIKE '%' || sqlc.arg(query) || '%')
  AND (NOT sqlc.arg(late_only)::boolean OR p.late_count > 0);

-- name: AddProjectMember :exec
INSERT INTO project_members (project_id, user_id, is_admin)
VALUES ($1, $2, $3);

-- name: GetProjectMembership :one
SELECT is_admin FROM project_members WHERE project_id = $1 AND user_id = $2;

-- name: ListProjectMembers :many
SELECT pm.user_id, u.name, u.email, pm.is_admin, pm.created_at
FROM project_members pm
JOIN users u ON u.id = pm.user_id
WHERE pm.project_id = $1
ORDER BY pm.created_at, u.name;

-- name: SetProjectMemberAdmin :execrows
UPDATE project_members SET is_admin = $3 WHERE project_id = $1 AND user_id = $2;

-- name: RemoveProjectMember :execrows
DELETE FROM project_members WHERE project_id = $1 AND user_id = $2;

-- name: LockProjectAdmins :many
SELECT user_id FROM project_members WHERE project_id = $1 AND is_admin FOR UPDATE;

-- name: AddProjectMemberIfAbsent :exec
INSERT INTO project_members (project_id, user_id, is_admin)
VALUES ($1, $2, $3)
ON CONFLICT (project_id, user_id) DO NOTHING;

-- name: FindUserByEmail :one
SELECT id, name, email FROM users WHERE email = $1;

-- name: UpdateProject :one
UPDATE projects
SET name = $2,
    description = $3,
    planned_start_date = $4,
    planned_end_date = $5,
    updated_at = NOW()
WHERE display_id = $1
RETURNING id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at, member_count, task_count_not_started, task_count_in_progress, task_count_done, late_count, blocked_count, progress;

-- name: DeleteProject :execrows
DELETE FROM projects WHERE display_id = $1;

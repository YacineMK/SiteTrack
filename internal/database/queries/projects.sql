-- name: CreateProject :one
INSERT INTO projects (id, display_id, name, description, planned_start_date, planned_end_date)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at;

-- name: GetProjectByDisplayID :one
SELECT id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at
FROM projects
WHERE display_id = $1;

-- name: ListProjects :many
WITH stats AS (
    SELECT project_id,
           COUNT(*) FILTER (WHERE status = 'not_started') AS not_started,
           COUNT(*) FILTER (WHERE status = 'in_progress') AS in_progress,
           COUNT(*) FILTER (WHERE status = 'done') AS done,
           COUNT(*) FILTER (WHERE (status = 'not_started' AND planned_start_date < CURRENT_DATE)
                              OR (status = 'in_progress' AND planned_end_date < CURRENT_DATE)) AS late
    FROM tasks
    GROUP BY project_id
), members AS (
    SELECT project_id, COUNT(*) AS member_count FROM project_members GROUP BY project_id
)
SELECT p.id, p.display_id, p.name, p.description, p.planned_start_date, p.planned_end_date,
       p.created_at, p.updated_at,
       EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)) AS is_admin,
       COALESCE(members.member_count, 0) AS member_count,
       COALESCE(stats.not_started, 0) AS not_started,
       COALESCE(stats.in_progress, 0) AS in_progress,
       COALESCE(stats.done, 0) AS done,
       COALESCE(stats.late, 0) AS late,
       CASE WHEN COALESCE(stats.not_started, 0) + COALESCE(stats.in_progress, 0) + COALESCE(stats.done, 0) = 0 THEN 0
            ELSE ROUND(COALESCE(stats.done, 0) * 100.0 / (COALESCE(stats.not_started, 0) + COALESCE(stats.in_progress, 0) + COALESCE(stats.done, 0)), 2)
       END::float8 AS progress
FROM projects p
LEFT JOIN stats ON stats.project_id = p.id
LEFT JOIN members ON members.project_id = p.id
WHERE (sqlc.arg(super_admin)::boolean
   OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)))
  AND (NOT sqlc.arg(late_only)::boolean OR COALESCE(stats.late, 0) > 0)
ORDER BY
  CASE WHEN sqlc.arg(sort_by)::text = 'name' AND sqlc.arg(sort_dir)::text = 'asc' THEN p.name END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'name' AND sqlc.arg(sort_dir)::text = 'desc' THEN p.name END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'start_date' AND sqlc.arg(sort_dir)::text = 'asc' THEN p.planned_start_date END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'start_date' AND sqlc.arg(sort_dir)::text = 'desc' THEN p.planned_start_date END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'progress' AND sqlc.arg(sort_dir)::text = 'asc' THEN
       CASE WHEN COALESCE(stats.not_started, 0) + COALESCE(stats.in_progress, 0) + COALESCE(stats.done, 0) = 0 THEN 0
            ELSE COALESCE(stats.done, 0)::float8 / (COALESCE(stats.not_started, 0) + COALESCE(stats.in_progress, 0) + COALESCE(stats.done, 0)) END
  END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'progress' AND sqlc.arg(sort_dir)::text = 'desc' THEN
       CASE WHEN COALESCE(stats.not_started, 0) + COALESCE(stats.in_progress, 0) + COALESCE(stats.done, 0) = 0 THEN 0
            ELSE COALESCE(stats.done, 0)::float8 / (COALESCE(stats.not_started, 0) + COALESCE(stats.in_progress, 0) + COALESCE(stats.done, 0)) END
  END DESC,
  CASE WHEN sqlc.arg(sort_by)::text = 'late' AND sqlc.arg(sort_dir)::text = 'asc' THEN COALESCE(stats.late, 0) END ASC,
  CASE WHEN sqlc.arg(sort_by)::text = 'late' AND sqlc.arg(sort_dir)::text = 'desc' THEN COALESCE(stats.late, 0) END DESC,
  p.created_at DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountProjects :one
WITH stats AS (
    SELECT project_id,
           COUNT(*) FILTER (WHERE status = 'not_started') AS not_started,
           COUNT(*) FILTER (WHERE status = 'in_progress') AS in_progress,
           COUNT(*) FILTER (WHERE status = 'done') AS done,
           COUNT(*) FILTER (WHERE (status = 'not_started' AND planned_start_date < CURRENT_DATE)
                              OR (status = 'in_progress' AND planned_end_date < CURRENT_DATE)) AS late
    FROM tasks
    GROUP BY project_id
), members AS (
    SELECT project_id, COUNT(*) AS member_count FROM project_members GROUP BY project_id
)
SELECT COUNT(*)
FROM projects p
LEFT JOIN stats ON stats.project_id = p.id
WHERE (sqlc.arg(super_admin)::boolean
   OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)))
  AND (NOT sqlc.arg(late_only)::boolean OR COALESCE(stats.late, 0) > 0);

-- name: SearchProjects :many
WITH stats AS (
    SELECT project_id,
           COUNT(*) FILTER (WHERE status = 'not_started') AS not_started,
           COUNT(*) FILTER (WHERE status = 'in_progress') AS in_progress,
           COUNT(*) FILTER (WHERE status = 'done') AS done,
           COUNT(*) FILTER (WHERE (status = 'not_started' AND planned_start_date < CURRENT_DATE)
                              OR (status = 'in_progress' AND planned_end_date < CURRENT_DATE)) AS late
    FROM tasks
    GROUP BY project_id
), members AS (
    SELECT project_id, COUNT(*) AS member_count FROM project_members GROUP BY project_id
)
SELECT p.id, p.display_id, p.name, p.description, p.planned_start_date, p.planned_end_date,
       p.created_at, p.updated_at,
       ts_rank_cd(p.search, websearch_to_tsquery('english', sqlc.arg(query))) AS rank,
       EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)) AS is_admin,
       COALESCE(members.member_count, 0) AS member_count,
       COALESCE(stats.not_started, 0) AS not_started,
       COALESCE(stats.in_progress, 0) AS in_progress,
       COALESCE(stats.done, 0) AS done,
       COALESCE(stats.late, 0) AS late,
       CASE WHEN COALESCE(stats.not_started, 0) + COALESCE(stats.in_progress, 0) + COALESCE(stats.done, 0) = 0 THEN 0
            ELSE ROUND(COALESCE(stats.done, 0) * 100.0 / (COALESCE(stats.not_started, 0) + COALESCE(stats.in_progress, 0) + COALESCE(stats.done, 0)), 2)
       END::float8 AS progress
FROM projects p
LEFT JOIN stats ON stats.project_id = p.id
LEFT JOIN members ON members.project_id = p.id
WHERE (sqlc.arg(super_admin)::boolean
   OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)))
  AND (p.search @@ websearch_to_tsquery('english', sqlc.arg(query))
   OR p.name ILIKE '%' || sqlc.arg(query) || '%'
   OR p.display_id ILIKE '%' || sqlc.arg(query) || '%')
  AND (NOT sqlc.arg(late_only)::boolean OR COALESCE(stats.late, 0) > 0)
ORDER BY rank DESC, p.created_at DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: CountSearchProjects :one
WITH stats AS (
    SELECT project_id,
           COUNT(*) FILTER (WHERE status = 'not_started') AS not_started,
           COUNT(*) FILTER (WHERE status = 'in_progress') AS in_progress,
           COUNT(*) FILTER (WHERE status = 'done') AS done,
           COUNT(*) FILTER (WHERE (status = 'not_started' AND planned_start_date < CURRENT_DATE)
                              OR (status = 'in_progress' AND planned_end_date < CURRENT_DATE)) AS late
    FROM tasks
    GROUP BY project_id
), members AS (
    SELECT project_id, COUNT(*) AS member_count FROM project_members GROUP BY project_id
)
SELECT COUNT(*)
FROM projects p
LEFT JOIN stats ON stats.project_id = p.id
WHERE (sqlc.arg(super_admin)::boolean
   OR EXISTS (SELECT 1 FROM project_members pm WHERE pm.project_id = p.id AND pm.user_id = sqlc.arg(user_id)))
  AND (p.search @@ websearch_to_tsquery('english', sqlc.arg(query))
   OR p.name ILIKE '%' || sqlc.arg(query) || '%'
   OR p.display_id ILIKE '%' || sqlc.arg(query) || '%')
  AND (NOT sqlc.arg(late_only)::boolean OR COALESCE(stats.late, 0) > 0);

-- name: ProjectBlockedCounts :many
WITH RECURSIVE reach AS (
    SELECT t.id AS task_id, t.project_id, td.depends_on AS upstream
    FROM tasks t
    JOIN task_dependencies td ON td.task_id = t.id
    WHERE t.project_id = ANY(sqlc.arg(project_ids)::uuid[])
    UNION
    SELECT r.task_id, r.project_id, td.depends_on
    FROM task_dependencies td
    JOIN reach r ON td.task_id = r.upstream
)
SELECT r.project_id, COUNT(DISTINCT r.task_id) AS blocked
FROM reach r
JOIN tasks u ON u.id = r.upstream
WHERE u.status <> 'done'
GROUP BY r.project_id;

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
RETURNING id, display_id, name, description, planned_start_date, planned_end_date, created_at, updated_at;

-- name: DeleteProject :execrows
DELETE FROM projects WHERE display_id = $1;

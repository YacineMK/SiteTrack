-- migrate:up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE tasks
    ADD COLUMN blocked BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN dependents_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE projects
    ADD COLUMN member_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN task_count_not_started INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN task_count_in_progress INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN task_count_done INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN late_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN blocked_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE projects
    ADD COLUMN progress FLOAT8 GENERATED ALWAYS AS (
        CASE
            WHEN task_count_not_started + task_count_in_progress + task_count_done = 0 THEN 0::float8
            ELSE round(task_count_done * 100.0 / (task_count_not_started + task_count_in_progress + task_count_done), 2)::float8
        END
    ) STORED;

CREATE OR REPLACE FUNCTION refresh_project_stats(p_project UUID) RETURNS void
LANGUAGE sql AS $$
    UPDATE projects p SET
        member_count = (SELECT COUNT(*)::int FROM project_members pm WHERE pm.project_id = p_project),
        task_count_not_started = s.not_started,
        task_count_in_progress = s.in_progress,
        task_count_done = s.done,
        late_count = s.late,
        blocked_count = s.blocked
    FROM (
        SELECT
            COUNT(*) FILTER (WHERE status = 'not_started')::int AS not_started,
            COUNT(*) FILTER (WHERE status = 'in_progress')::int AS in_progress,
            COUNT(*) FILTER (WHERE status = 'done')::int AS done,
            COUNT(*) FILTER (WHERE (status = 'not_started' AND planned_start_date < CURRENT_DATE)
                               OR (status = 'in_progress' AND planned_end_date < CURRENT_DATE))::int AS late,
            COUNT(*) FILTER (WHERE blocked)::int AS blocked
        FROM tasks
        WHERE project_id = p_project
    ) s
    WHERE p.id = p_project;
$$;

CREATE OR REPLACE FUNCTION recompute_blocked(p_task UUID) RETURNS void
LANGUAGE sql AS $$
    WITH RECURSIVE affected AS (
        SELECT p_task AS id
        UNION
        SELECT td.task_id FROM task_dependencies td JOIN affected a ON td.depends_on = a.id
    ), blocked AS (
        SELECT DISTINCT td.task_id AS id
        FROM task_dependencies td
        JOIN affected a ON a.id = td.task_id
        JOIN tasks u ON u.id = td.depends_on
        WHERE u.status <> 'done' OR u.blocked
        UNION
        SELECT td.task_id
        FROM task_dependencies td
        JOIN blocked b ON td.depends_on = b.id
        JOIN affected a ON a.id = td.task_id
    )
    UPDATE tasks t
    SET blocked = (t.id IN (SELECT id FROM blocked))
    WHERE t.id IN (SELECT id FROM affected)
      AND t.blocked IS DISTINCT FROM (t.id IN (SELECT id FROM blocked));
$$;

UPDATE tasks t SET dependents_count = c.n
FROM (SELECT depends_on AS id, COUNT(*)::int AS n FROM task_dependencies GROUP BY depends_on) c
WHERE t.id = c.id;

DO $$
DECLARE added INTEGER;
BEGIN
    CREATE TEMP TABLE _blocked (id UUID PRIMARY KEY);
    INSERT INTO _blocked (id)
    SELECT DISTINCT td.task_id
    FROM task_dependencies td
    JOIN tasks u ON u.id = td.depends_on
    WHERE u.status <> 'done';
    LOOP
        INSERT INTO _blocked (id)
        SELECT DISTINCT td.task_id
        FROM task_dependencies td
        JOIN _blocked b ON td.depends_on = b.id
        ON CONFLICT DO NOTHING;
        GET DIAGNOSTICS added = ROW_COUNT;
        EXIT WHEN added = 0;
    END LOOP;
    UPDATE tasks t SET blocked = TRUE FROM _blocked b WHERE t.id = b.id AND NOT t.blocked;
    DROP TABLE _blocked;
END $$;

SELECT refresh_project_stats(id) FROM projects;

CREATE OR REPLACE FUNCTION tasks_after_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refresh_project_stats(OLD.project_id);
        RETURN OLD;
    END IF;

    IF TG_OP = 'UPDATE' AND NEW.status IS DISTINCT FROM OLD.status THEN
        PERFORM recompute_blocked(NEW.id);
    END IF;

    PERFORM refresh_project_stats(NEW.project_id);
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION task_dependencies_after_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE tasks SET dependents_count = dependents_count + 1 WHERE id = NEW.depends_on;
        PERFORM recompute_blocked(NEW.depends_on);
        PERFORM refresh_project_stats(pid) FROM (
            SELECT DISTINCT project_id AS pid FROM tasks WHERE id IN (NEW.task_id, NEW.depends_on)
        ) s;
        RETURN NEW;
    END IF;

    UPDATE tasks SET dependents_count = GREATEST(dependents_count - 1, 0) WHERE id = OLD.depends_on;
    PERFORM recompute_blocked(OLD.depends_on);
    PERFORM recompute_blocked(OLD.task_id);
    PERFORM refresh_project_stats(pid) FROM (
        SELECT DISTINCT project_id AS pid FROM tasks WHERE id IN (OLD.task_id, OLD.depends_on)
    ) s;
    RETURN OLD;
END;
$$;

CREATE OR REPLACE FUNCTION project_members_after_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        PERFORM refresh_project_stats(OLD.project_id);
        RETURN OLD;
    END IF;
    PERFORM refresh_project_stats(NEW.project_id);
    RETURN NEW;
END;
$$;

CREATE TRIGGER tasks_after_change
    AFTER INSERT OR DELETE OR UPDATE OF status, planned_start_date, planned_end_date
    ON tasks FOR EACH ROW EXECUTE FUNCTION tasks_after_change();

CREATE TRIGGER task_dependencies_after_change
    AFTER INSERT OR DELETE
    ON task_dependencies FOR EACH ROW EXECUTE FUNCTION task_dependencies_after_change();

CREATE TRIGGER project_members_after_change
    AFTER INSERT OR DELETE
    ON project_members FOR EACH ROW EXECUTE FUNCTION project_members_after_change();

CREATE INDEX tasks_project_display_idx ON tasks (project_id, display_id);
CREATE INDEX tasks_project_name_idx ON tasks (project_id, name, display_id);
CREATE INDEX tasks_project_start_idx ON tasks (project_id, planned_start_date, display_id);
CREATE INDEX tasks_project_end_idx ON tasks (project_id, planned_end_date, display_id);
CREATE INDEX tasks_project_dependents_idx ON tasks (project_id, dependents_count, display_id);
CREATE INDEX tasks_project_status_idx ON tasks (project_id, status);
CREATE INDEX tasks_project_blocked_idx ON tasks (project_id) WHERE blocked;
CREATE INDEX tasks_name_trgm_idx ON tasks USING GIN (name gin_trgm_ops);

CREATE INDEX projects_created_idx ON projects (created_at DESC, id DESC);
CREATE INDEX projects_name_sorted_idx ON projects (name, id);
CREATE INDEX projects_start_sorted_idx ON projects (planned_start_date, id);
CREATE INDEX projects_progress_sorted_idx ON projects (progress, id);
CREATE INDEX projects_late_sorted_idx ON projects (late_count, id);

CREATE INDEX deliveries_sent_idx ON deliveries (sent_at DESC, id DESC);
CREATE INDEX deliveries_status_sent_idx ON deliveries (status, sent_at DESC, id DESC);
CREATE INDEX deliveries_sender_sent_idx ON deliveries (sender_type, sender_id, sent_at DESC, id DESC);
CREATE INDEX deliveries_receiver_sent_idx ON deliveries (receiver_type, receiver_id, sent_at DESC, id DESC);

-- migrate:down
DROP INDEX deliveries_receiver_sent_idx;
DROP INDEX deliveries_sender_sent_idx;
DROP INDEX deliveries_status_sent_idx;
DROP INDEX deliveries_sent_idx;
DROP INDEX projects_late_sorted_idx;
DROP INDEX projects_progress_sorted_idx;
DROP INDEX projects_start_sorted_idx;
DROP INDEX projects_name_sorted_idx;
DROP INDEX projects_created_idx;
DROP INDEX tasks_name_trgm_idx;
DROP INDEX tasks_project_blocked_idx;
DROP INDEX tasks_project_status_idx;
DROP INDEX tasks_project_dependents_idx;
DROP INDEX tasks_project_end_idx;
DROP INDEX tasks_project_start_idx;
DROP INDEX tasks_project_name_idx;
DROP INDEX tasks_project_display_idx;

DROP TRIGGER project_members_after_change ON project_members;
DROP TRIGGER task_dependencies_after_change ON task_dependencies;
DROP TRIGGER tasks_after_change ON tasks;

DROP FUNCTION project_members_after_change();
DROP FUNCTION task_dependencies_after_change();
DROP FUNCTION tasks_after_change();
DROP FUNCTION recompute_blocked(UUID);
DROP FUNCTION refresh_project_stats(UUID);

ALTER TABLE projects
    DROP COLUMN progress,
    DROP COLUMN blocked_count,
    DROP COLUMN late_count,
    DROP COLUMN task_count_done,
    DROP COLUMN task_count_in_progress,
    DROP COLUMN task_count_not_started,
    DROP COLUMN member_count;

ALTER TABLE tasks
    DROP COLUMN dependents_count,
    DROP COLUMN blocked;

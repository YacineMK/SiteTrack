-- migrate:up
CREATE TABLE task_id_counter (
    project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    last_value BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE tasks (
    id                 UUID PRIMARY KEY,
    display_id         TEXT NOT NULL UNIQUE,
    project_id         UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name               TEXT NOT NULL,
    planned_start_date DATE NOT NULL,
    planned_end_date   DATE NOT NULL,
    status             TEXT NOT NULL DEFAULT 'not_started'
                       CHECK (status IN ('not_started', 'in_progress', 'done')),
    blocked            BOOLEAN NOT NULL DEFAULT FALSE,
    dependents_count   INTEGER NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX tasks_project_name_idx    ON tasks (project_id, name, id);
CREATE INDEX tasks_project_start_idx   ON tasks (project_id, planned_start_date, id);
CREATE INDEX tasks_project_end_idx     ON tasks (project_id, planned_end_date, id);
CREATE INDEX tasks_project_deps_idx    ON tasks (project_id, dependents_count DESC, id);
CREATE INDEX tasks_project_status_idx  ON tasks (project_id, status);
CREATE INDEX tasks_project_blocked_idx ON tasks (project_id) WHERE blocked;

CREATE TABLE task_dependencies (
    task_id    UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, depends_on),
    CHECK (task_id <> depends_on)
);

CREATE INDEX task_dependencies_depends_on_idx ON task_dependencies (depends_on);

-- blocked = any upstream not done, transitively. recomputed on dep change or status change.
CREATE FUNCTION recompute_blocked(root UUID) RETURNS void LANGUAGE sql AS $$
    WITH RECURSIVE reach(id) AS (
        SELECT root
        UNION
        SELECT td.task_id FROM task_dependencies td JOIN reach ON td.depends_on = reach.id
    ),
    want(id, blocked) AS (
        SELECT r.id, EXISTS (
            WITH RECURSIVE up(id) AS (
                SELECT td.depends_on FROM task_dependencies td WHERE td.task_id = r.id
                UNION
                SELECT td.depends_on FROM task_dependencies td JOIN up ON td.task_id = up.id
            )
            SELECT 1 FROM up JOIN tasks ON tasks.id = up.id WHERE tasks.status <> 'done'
        )
        FROM reach r
    )
    UPDATE tasks t SET blocked = w.blocked
    FROM want w
    WHERE t.id = w.id AND t.blocked IS DISTINCT FROM w.blocked;
$$;

CREATE FUNCTION tasks_count_bump(pid UUID, status TEXT, delta INT) RETURNS void LANGUAGE sql AS $$
    UPDATE projects SET
        task_count_not_started = task_count_not_started + CASE status WHEN 'not_started' THEN delta ELSE 0 END,
        task_count_in_progress = task_count_in_progress + CASE status WHEN 'in_progress' THEN delta ELSE 0 END,
        task_count_done        = task_count_done        + CASE status WHEN 'done'        THEN delta ELSE 0 END,
        updated_at = NOW()
    WHERE id = pid;
$$;

CREATE FUNCTION tasks_after_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM tasks_count_bump(NEW.project_id, NEW.status, 1);
    ELSIF TG_OP = 'DELETE' THEN
        PERFORM tasks_count_bump(OLD.project_id, OLD.status, -1);
    ELSIF NEW.status IS DISTINCT FROM OLD.status THEN
        PERFORM tasks_count_bump(NEW.project_id, OLD.status, -1);
        PERFORM tasks_count_bump(NEW.project_id, NEW.status, 1);
        PERFORM recompute_blocked(NEW.id);
    END IF;
    RETURN NULL;
END $$;

CREATE TRIGGER tasks_after_change
AFTER INSERT OR UPDATE OF status OR DELETE ON tasks
FOR EACH ROW EXECUTE FUNCTION tasks_after_change();

CREATE FUNCTION task_deps_after_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE tasks SET dependents_count = dependents_count + 1 WHERE id = NEW.depends_on;
        PERFORM recompute_blocked(NEW.task_id);
    ELSE
        UPDATE tasks SET dependents_count = GREATEST(dependents_count - 1, 0) WHERE id = OLD.depends_on;
        PERFORM recompute_blocked(OLD.task_id);
    END IF;
    RETURN NULL;
END $$;

CREATE TRIGGER task_deps_after_change
AFTER INSERT OR DELETE ON task_dependencies
FOR EACH ROW EXECUTE FUNCTION task_deps_after_change();

-- migrate:down
DROP TABLE task_dependencies;
DROP TABLE tasks;
DROP TABLE task_id_counter;
DROP FUNCTION task_deps_after_change;
DROP FUNCTION tasks_after_change;
DROP FUNCTION tasks_count_bump;
DROP FUNCTION recompute_blocked;

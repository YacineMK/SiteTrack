-- migrate:up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE project_id_counter (
    id         BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    last_value BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE projects (
    id                     UUID PRIMARY KEY,
    display_id             TEXT NOT NULL UNIQUE,
    name                   TEXT NOT NULL,
    description            TEXT NOT NULL DEFAULT '',
    planned_start_date     DATE NOT NULL,
    planned_end_date       DATE NOT NULL,
    member_count           INTEGER NOT NULL DEFAULT 0,
    task_count_not_started INTEGER NOT NULL DEFAULT 0,
    task_count_in_progress INTEGER NOT NULL DEFAULT 0,
    task_count_done        INTEGER NOT NULL DEFAULT 0,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX projects_name_idx    ON projects (name, id);
CREATE INDEX projects_start_idx   ON projects (planned_start_date, id);
CREATE INDEX projects_created_idx ON projects (created_at DESC, id DESC);
CREATE INDEX projects_name_trgm_idx       ON projects USING GIN (name gin_trgm_ops);
CREATE INDEX projects_display_id_trgm_idx ON projects USING GIN (display_id gin_trgm_ops);

CREATE TABLE project_members (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    is_admin   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX project_members_user_idx ON project_members (user_id);

CREATE FUNCTION project_members_count() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    UPDATE projects SET member_count = member_count + CASE TG_OP WHEN 'INSERT' THEN 1 ELSE -1 END
    WHERE id = COALESCE(NEW.project_id, OLD.project_id);
    RETURN NULL;
END $$;

CREATE TRIGGER project_members_count
AFTER INSERT OR DELETE ON project_members
FOR EACH ROW EXECUTE FUNCTION project_members_count();

-- migrate:down
DROP TABLE project_members;
DROP FUNCTION project_members_count;
DROP TABLE projects;
DROP TABLE project_id_counter;

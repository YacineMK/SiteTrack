-- migrate:up
CREATE TABLE tasks (
    id                 UUID PRIMARY KEY,
    display_id         TEXT NOT NULL,
    project_id         UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name               TEXT NOT NULL,
    planned_start_date DATE NOT NULL,
    planned_end_date   DATE NOT NULL,
    status             TEXT NOT NULL DEFAULT 'not_started',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX tasks_display_id_idx ON tasks (display_id);
CREATE INDEX tasks_project_idx ON tasks (project_id);
CREATE INDEX tasks_status_idx ON tasks (status);

CREATE TABLE task_id_counter (
    project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    last_value BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE task_dependencies (
    task_id    UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, depends_on),
    CHECK (task_id <> depends_on)
);

CREATE INDEX task_dependencies_depends_on_idx ON task_dependencies (depends_on);

-- migrate:down
DROP TABLE task_dependencies;
DROP TABLE task_id_counter;
DROP TABLE tasks;

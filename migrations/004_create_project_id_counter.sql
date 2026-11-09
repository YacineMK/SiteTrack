-- migrate:up
CREATE TABLE project_id_counter (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE,
    last_value BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT project_id_counter_single_row CHECK (id)
);

-- migrate:down
DROP TABLE project_id_counter;

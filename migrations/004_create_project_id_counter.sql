-- +goose Up
CREATE TABLE project_id_counter (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE,
    last_value BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT project_id_counter_single_row CHECK (id)
);

-- +goose Down
DROP TABLE project_id_counter;

-- +goose Up
ALTER TABLE deliveries ADD COLUMN sent_by UUID REFERENCES users(id);
ALTER TABLE deliveries ADD COLUMN received_by UUID REFERENCES users(id);

-- +goose Down
ALTER TABLE deliveries DROP COLUMN received_by;
ALTER TABLE deliveries DROP COLUMN sent_by;

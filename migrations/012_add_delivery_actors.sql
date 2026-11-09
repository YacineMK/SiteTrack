-- migrate:up
ALTER TABLE deliveries ADD COLUMN sent_by UUID REFERENCES users(id);
ALTER TABLE deliveries ADD COLUMN received_by UUID REFERENCES users(id);

-- migrate:down
ALTER TABLE deliveries DROP COLUMN received_by;
ALTER TABLE deliveries DROP COLUMN sent_by;

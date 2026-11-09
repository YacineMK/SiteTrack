-- +goose Up
CREATE TABLE materials (
    id         UUID PRIMARY KEY,
    name       TEXT NOT NULL,
    category   TEXT NOT NULL,
    unit       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX materials_name_idx ON materials (LOWER(name));
CREATE INDEX materials_category_idx ON materials (category);

CREATE TABLE company_store_stock (
    material_id   UUID PRIMARY KEY REFERENCES materials(id) ON DELETE CASCADE,
    quantity      NUMERIC(14,3) NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    reorder_point NUMERIC(14,3) NOT NULL DEFAULT 0 CHECK (reorder_point >= 0),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE project_store_stock (
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    material_id UUID NOT NULL REFERENCES materials(id) ON DELETE CASCADE,
    quantity    NUMERIC(14,3) NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project_id, material_id)
);

CREATE TYPE delivery_party AS ENUM ('company', 'project');
CREATE TYPE delivery_status AS ENUM ('in_transit', 'delivered', 'rejected', 'cancelled');

CREATE TABLE deliveries (
    id            UUID PRIMARY KEY,
    sender_type   delivery_party NOT NULL,
    sender_id     UUID,
    receiver_type delivery_party NOT NULL,
    receiver_id   UUID,
    status        delivery_status NOT NULL DEFAULT 'in_transit',
    sent_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    received_at   TIMESTAMPTZ,
    CHECK ((sender_type = 'company') = (sender_id IS NULL)),
    CHECK ((receiver_type = 'company') = (receiver_id IS NULL)),
    CHECK (sender_type <> receiver_type)
);

CREATE INDEX deliveries_sender_idx ON deliveries (sender_id);
CREATE INDEX deliveries_receiver_idx ON deliveries (receiver_id);
CREATE INDEX deliveries_status_idx ON deliveries (status);

CREATE TABLE delivery_lines (
    id          UUID PRIMARY KEY,
    delivery_id UUID NOT NULL REFERENCES deliveries(id) ON DELETE CASCADE,
    material_id UUID NOT NULL REFERENCES materials(id),
    quantity    NUMERIC(14,3) NOT NULL CHECK (quantity > 0),
    UNIQUE (delivery_id, material_id)
);

CREATE INDEX delivery_lines_delivery_idx ON delivery_lines (delivery_id);

-- +goose Down
DROP TABLE delivery_lines;
DROP TABLE deliveries;
DROP TYPE delivery_status;
DROP TYPE delivery_party;
DROP TABLE project_store_stock;
DROP TABLE company_store_stock;
DROP TABLE materials;

-- +goose Up
CREATE TABLE requests (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    handler TEXT NOT NULL,
    headers JSONB NOT NULL,
    body JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- links between tables are not used to improve performance
CREATE TABLE request_collections (
    request_id BIGINT NOT NULL,
    collection_id BIGINT NOT NULL,
    PRIMARY KEY (request_id, collection_id)
);

CREATE INDEX idx_requests_created_at ON requests(created_at);
CREATE INDEX idx_request_collections_request_id ON request_collections(request_id);
CREATE INDEX idx_request_collections_collection_id ON request_collections(collection_id);

-- +goose Down
DROP TABLE request_collections;
DROP TABLE requests;

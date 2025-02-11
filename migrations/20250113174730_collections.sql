-- +goose Up

CREATE TABLE collections (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    status INTEGER NOT NULL,    
    request_count_limit INTEGER NOT NULL,
    request_duration_limit INTERVAL NOT NULL,
    criteria JSONB NOT NULL,
    request_count INTEGER NOT NULL DEFAULT 0,    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    result_id TEXT,
    error_message TEXT,
    error_code INTEGER
);

CREATE INDEX idx_collections_status ON collections(status);
CREATE INDEX idx_collections_created_at ON collections(created_at);
CREATE INDEX idx_collections_completed_at ON collections(completed_at);

-- +goose Down

DROP TABLE IF EXISTS collections;

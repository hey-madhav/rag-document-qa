CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS chunks (
    id             BIGSERIAL PRIMARY KEY,
    doc_name       TEXT        NOT NULL,
    chunk_text     TEXT        NOT NULL,
    token_count    INT         NOT NULL,
    chunk_strategy VARCHAR(64) NOT NULL,
    embedding      vector(1536) NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS chunks_embedding_ivfflat
    ON chunks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX IF NOT EXISTS chunks_chunk_strategy_idx ON chunks (chunk_strategy);
CREATE INDEX IF NOT EXISTS chunks_doc_name_idx ON chunks (doc_name);

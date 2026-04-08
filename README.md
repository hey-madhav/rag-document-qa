# rag-document-qa

RAG-powered Document Q&A API in Go (Gin + pgvector + Gemini).

This project is a backend service that turns raw document text into searchable knowledge and answers user questions with grounded context. It is designed as a practical, production-oriented RAG reference in idiomatic Go: clear package boundaries, dependency inversion via small interfaces, and simple deployment as a single binary.

At a high level, the API supports two flows:
- Ingestion: split a document into chunks, generate embeddings, and store both text and vectors in PostgreSQL with `pgvector`.
- Querying: embed the incoming question, retrieve the nearest chunks, and pass them as context to Gemini to generate an answer.

The goal is to balance correctness, maintainability, and operational simplicity. Instead of introducing many moving parts, the stack keeps retrieval and metadata in the same database, exposes a minimal HTTP surface with Gin, and keeps room for future upgrades such as reranking, hybrid search, and caching.

## Endpoints

`POST /ingest`

Request body:
```json
{
  "doc_name": "policy.pdf",
  "content": "Full document text...",
  "chunk_strategy": "token_256"
}
```

`POST /ask`

Request body:
```json
{
  "query": "What is the return policy?",
  "top_k": 4,
  "chunk_strategy": "token_256"
}
```

## Local setup

1. Copy `.env.example` to `.env` and set values.
2. Run migrations:
   - Railway runs `psql $DATABASE_URL -f migrations/schema.sql` via the `release:` Procfile line.
   - Locally, run: `psql "$DATABASE_URL" -f migrations/schema.sql`
3. Start the API:
   - `go run ./cmd/server`

## Docker setup

Build the image:

```bash
docker build -t rag-document-qa .
```

Run the container:

```bash
docker run --rm -p 8080:8080 -v "$(pwd)/config.yml:/app/config.yml:ro" rag-document-qa
```

If your config file lives elsewhere, pass it explicitly:

```bash
docker run --rm -p 8080:8080 -e CONFIG_FILE=/app/config/custom.yml -v "$(pwd)/config.yml:/app/config/custom.yml:ro" rag-document-qa
```

### Engineering Decisions
**Why Go over Python**
Single binary deployment, no virtualenv, built-in concurrency for batching embedding requests across goroutines, and interfaces that enforce dependency inversion at compile time rather than runtime.

**Chunking strategy — 256 vs 512 tokens**

| Strategy   | Avg Hit Rate | Avg Latency | Notes                                |
|------------|-------------|-------------|---------------------------------------|
| token_256  | —           | —           | Fill from experiment_results.json     |
| token_512  | —           | —           | Fill from experiment_results.json     |

Decision: 256-token chunks with 20-token overlap as default. Rationale: factual Q&A rewards precision over context breadth; overlap preserves sentence continuity at boundaries.

**Interfaces defined by consumers, not implementors**
Go interfaces are satisfied implicitly. `ChunkRepository` is defined in the `storage` package (consumer), not in a separate `interfaces` package. This follows the Go proverb: accept interfaces, return structs. It means adding a new vector store requires zero changes to business logic.

**pgvector over Pinecone**
One fewer managed service, SQL filtering by `doc_name` and `chunk_strategy` for free, co-located with app data. At portfolio scale, operational simplicity beats ANN performance.

**What I'd change at scale**
- Worker pool with bounded goroutines for large document ingestion
- Cross-encoder reranking via a local `ms-marco-MiniLM` model (implements `Reranker` interface — already stubbed)
- Redis query cache with `sync.Map` as a local fallback
- Hybrid search: BM25 + vector with reciprocal rank fusion

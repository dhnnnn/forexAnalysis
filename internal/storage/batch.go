package storage

import "github.com/jackc/pgx/v5"

// pgxBatch wraps pgx.Batch for convenience.
type pgxBatch struct {
	b pgx.Batch
}

// Queue adds a query to the batch.
func (pb *pgxBatch) Queue(query string, args ...any) {
	pb.b.Queue(query, args...)
}

// batch returns the underlying pgx.Batch.
func (pb *pgxBatch) batch() *pgx.Batch {
	return &pb.b
}

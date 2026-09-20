package persistence

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// A contractor with a valuation behind it cannot be deleted — migration 023
// made that a RESTRICT foreign key — and the service used to answer that
// with a bare 500. The refusal is the caller's data, so it is a conflict.
func TestRefusedByReference(t *testing.T) {
	fk := &pgconn.PgError{Code: "23503", Message: "violates foreign key constraint"}
	if got := refusedByReference(fk); !errors.Is(got, ErrGovInUse) {
		t.Fatalf("23503 → %v, want ErrGovInUse", got)
	}
	unique := &pgconn.PgError{Code: "23505"}
	if got := refusedByReference(unique); got != unique {
		t.Fatalf("23505 must pass through, got %v", got)
	}
	other := errors.New("connection reset")
	if got := refusedByReference(other); got != other {
		t.Fatalf("plain error must pass through, got %v", got)
	}
	if refusedByReference(nil) != nil {
		t.Fatal("nil must stay nil")
	}
}

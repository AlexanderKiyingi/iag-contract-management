package persistence_test

// Integration tests against a real PostgreSQL instance. Run with:
//
//   TEST_DATABASE_URL=postgres://cm:cm@localhost:5434/cm?sslmode=disable \
//     go test ./internal/persistence/... -run Integration -v
//
// Skip the file entirely if TEST_DATABASE_URL is unset so unit-test runs
// (e.g. CI without a Postgres) still pass.

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
)

func testDB(t *testing.T) (*persistence.Postgres, func()) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pg, err := persistence.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Wipe operational tables so tests are reproducible. The schema
	// migrations have already run as part of Connect.
	tables := []string{
		"task_items", "task_projects", "audit_entries", "assistance_messages",
		"profile_photos", "materials", "milestones", "custom_roles",
		"contracts", "engineers", "contractors", "workspace_users", "zones",
		"app_meta", "contractor_supervisors",
	}
	for _, tbl := range tables {
		if _, err := pg.Pool.Exec(ctx, "TRUNCATE TABLE "+tbl+" RESTART IDENTITY CASCADE"); err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}
	return pg, func() { pg.Close() }
}

// TestIntegration_ConcurrentContractPatches verifies the old data-loss bug
// is fixed: 50 goroutines patching the same contract in parallel should
// result in 50 successful patches, a single contract row in the DB, and the
// final state should be one of the 50 candidates (not a lost-update default).
//
// Pre-fix behaviour: each patch called SaveState which DELETE'd every row
// and reinserted from in-memory state; concurrent writers raced and most
// updates vanished.
func TestIntegration_ConcurrentContractPatches(t *testing.T) {
	pg, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	// Seed one zone (FK target) and one contract.
	if _, err := pg.Pool.Exec(ctx, `
		INSERT INTO zones (code, name) VALUES ('Z1', 'Zone 1')`); err != nil {
		t.Fatalf("seed zone: %v", err)
	}
	contract := models.Contract{
		No: "C-0001", Name: "Init", Zone: "Z1", Cs: 1000, Paid: 0, Bal: 1000,
		Prog: 0, Status: models.StatusPlanning, Pri: models.PriorityMedium,
		Workers: 1, Created: time.Now().Format("2006-01-02"),
	}
	if err := pg.InsertContract(ctx, contract); err != nil {
		t.Fatalf("insert contract: %v", err)
	}

	const N = 50
	var wg sync.WaitGroup
	errs := make(chan error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := contract
			c.Remarks = fmt.Sprintf("patch-%02d", i)
			c.Prog = i
			if err := pg.UpdateContract(ctx, c); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent patch failed: %v", err)
	}

	// DB must still contain exactly one row, with a remarks value from one
	// of the 50 patches (we can't predict which won, but it must be a real
	// one, not the "" we started with — i.e. at least one write landed).
	var count int
	if err := pg.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM contracts`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 contract row, got %d", count)
	}
	var remarks string
	if err := pg.Pool.QueryRow(ctx,
		`SELECT remarks FROM contracts WHERE contract_no='C-0001'`).Scan(&remarks); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if len(remarks) < len("patch-00") || remarks[:6] != "patch-" {
		t.Fatalf("final remarks not one of the patches: %q", remarks)
	}
}

// TestIntegration_AuditTrimKeepsMostRecent confirms InsertAuditAndTrim caps
// the table at the configured retention and keeps the most-recent rows.
func TestIntegration_AuditTrimKeepsMostRecent(t *testing.T) {
	pg, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	const keep = 10
	for i := 0; i < 25; i++ {
		entry := models.AuditEntry{
			ID:     fmt.Sprintf("AUD-%03d", i),
			At:     time.Now().Format("2006-01-02 15:04"),
			User:   "tester",
			Action: "test",
			Detail: fmt.Sprintf("entry-%02d", i),
		}
		if err := pg.InsertAuditAndTrim(ctx, entry, keep); err != nil {
			t.Fatalf("insert audit %d: %v", i, err)
		}
		// Microscopic sleep so logged_at_ts (default NOW()) differs between
		// inserts; otherwise the trim's ORDER BY can't break ties
		// deterministically.
		time.Sleep(2 * time.Millisecond)
	}

	var count int
	if err := pg.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_entries`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != keep {
		t.Fatalf("expected %d audit rows after trim, got %d", keep, count)
	}
	// Verify the OLDEST surviving row is from the second half (i.e. early
	// entries were correctly evicted).
	var oldestDetail string
	if err := pg.Pool.QueryRow(ctx,
		`SELECT detail FROM audit_entries ORDER BY logged_at_ts ASC LIMIT 1`).Scan(&oldestDetail); err != nil {
		t.Fatalf("read oldest: %v", err)
	}
	if oldestDetail == "entry-00" {
		t.Fatalf("trim did not evict oldest entries; got %q", oldestDetail)
	}
}

// TestIntegration_DeleteCustomRoleClearsAssignments verifies the role-delete
// transaction also nulls workspace_users.custom_role_id atomically.
func TestIntegration_DeleteCustomRoleClearsAssignments(t *testing.T) {
	pg, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()

	roleID := "ROLE-001"
	if err := pg.InsertCustomRole(ctx, models.CustomRole{
		ID:          roleID,
		Name:        "Tester",
		Description: "",
		Permissions: []string{"contracts.read"},
	}); err != nil {
		t.Fatalf("insert role: %v", err)
	}
	if err := pg.InsertWorkspaceUser(ctx, models.WorkspaceUser{
		ID: "USR-001", Email: "t@x.test", DisplayName: "T",
		Role: "viewer", Status: "active", CustomRoleID: &roleID,
	}); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if err := pg.DeleteCustomRole(ctx, roleID); err != nil {
		t.Fatalf("delete role: %v", err)
	}
	var assignedRole *string
	if err := pg.Pool.QueryRow(ctx,
		`SELECT custom_role_id FROM workspace_users WHERE id='USR-001'`).Scan(&assignedRole); err != nil {
		t.Fatalf("read user: %v", err)
	}
	if assignedRole != nil {
		t.Fatalf("custom_role_id was not cleared, still %q", *assignedRole)
	}
}

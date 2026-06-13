package persistence_test

// Governance smoke test against a real PostgreSQL instance. Exercises the
// gov_* schema (migrations 005-007) and the GovStore persistence layer end to
// end: contract -> milestone -> payment workflow -> variation workflow ->
// requisition + value-banded approval routing. Run with:
//
//   TEST_DATABASE_URL=postgres://cm@127.0.0.1:5499/cm?sslmode=disable \
//     go test ./internal/persistence/... -run GovernanceSmoke -v
//
// Skipped when TEST_DATABASE_URL is unset, like the other integration tests.

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alvor-technologies/iag-contract-management/internal/models"
	"github.com/alvor-technologies/iag-contract-management/internal/persistence"
)

func govTestStore(t *testing.T) (*persistence.GovStore, context.Context, func()) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping governance smoke test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	pg, err := persistence.Connect(ctx, url)
	if err != nil {
		cancel()
		t.Fatalf("connect: %v", err)
	}
	govTables := []string{
		"gov_payments", "gov_variations", "gov_requisitions", "gov_milestones",
		"gov_obligations", "gov_approval_rules", "gov_templates", "gov_clauses",
		"gov_budgets", "gov_closeouts", "gov_contracts",
	}
	for _, tbl := range govTables {
		if _, err := pg.Pool.Exec(ctx, "TRUNCATE TABLE "+tbl+" RESTART IDENTITY CASCADE"); err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}
	return persistence.NewGovStore(pg.Pool), ctx, func() { pg.Close(); cancel() }
}

func TestGovernanceSmoke_FullWorkflow(t *testing.T) {
	gs, ctx, cleanup := govTestStore(t)
	defer cleanup()

	// --- Contract ---
	c, err := gs.CreateContract(ctx, models.GovContract{
		ID:         models.NewGovID("GCT"),
		Number:     "GC-SMOKE-001",
		Name:       "Smoke Test Works",
		Contractor: "Acme Builders",
		Value:      100_000_000,
		Retention:  10,
		Status:     models.GovDraft,
		Activity:   []models.GovActivity{{Date: "now", Actor: "tester", Action: "created"}},
	})
	if err != nil {
		t.Fatalf("CreateContract: %v", err)
	}
	if got, err := gs.GetContract(ctx, "GC-SMOKE-001"); err != nil || got.ID != c.ID {
		t.Fatalf("GetContract by number: got %+v err %v", got, err)
	}

	// --- Milestone ---
	m, err := gs.CreateMilestone(ctx, models.GovMilestone{
		ID: models.NewGovID("GMS"), ContractID: c.ID, Name: "Foundations",
		Value: 20_000_000, Status: models.MSPending,
		Checklist: []models.ChecklistItem{{Item: "Excavation", Done: true}},
	})
	if err != nil {
		t.Fatalf("CreateMilestone: %v", err)
	}

	// --- Payment workflow: PM -> Finance -> Authorization -> Paid ---
	p := models.NewPayment(models.NewGovID("GPAY"), m.ID, c.ID, m.Value, c.Retention)
	wantPayable := int64(18_000_000) // 20M * (100-10)/100
	if p.Payable != wantPayable {
		t.Fatalf("payable: got %d want %d", p.Payable, wantPayable)
	}
	if _, err := gs.CreatePayment(ctx, p); err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	stored, _ := gs.GetPayment(ctx, p.ID)
	var authorizedAt int
	for i := 1; i <= 4; i++ {
		_, authorized, paid, aerr := stored.Advance("approver", "now")
		if aerr != nil {
			t.Fatalf("payment advance %d: %v", i, aerr)
		}
		if _, err := gs.UpdatePayment(ctx, *stored); err != nil {
			t.Fatalf("UpdatePayment %d: %v", i, err)
		}
		if authorized && authorizedAt == 0 {
			authorizedAt = i
		}
		if paid && i != 4 {
			t.Fatalf("payment marked paid at step %d, expected 4", i)
		}
	}
	if authorizedAt != 3 {
		t.Fatalf("payment authorized at step %d, expected 3 (Payment Authorization)", authorizedAt)
	}
	// Round-trips by milestone (used by the idempotent CreatePayment path).
	if bm, err := gs.GetPaymentByMilestone(ctx, m.ID); err != nil || bm.ID != p.ID {
		t.Fatalf("GetPaymentByMilestone: got %+v err %v", bm, err)
	}

	// --- Variation workflow: 4 approvals then contract value adjusts ---
	v := models.NewVariation(models.NewGovID("GVAR"), c.ID, "VO-1", "Extra works",
		5_000_000, 14, "scope add", "client request", "schedule", "pm", "now")
	if _, err := gs.CreateVariation(ctx, v); err != nil {
		t.Fatalf("CreateVariation: %v", err)
	}
	sv, _ := gs.GetVariation(ctx, v.ID)
	var approved bool
	for i := 1; i <= 4 && !approved; i++ {
		a, aerr := sv.Advance("approver", "now")
		if aerr != nil {
			t.Fatalf("variation advance %d: %v", i, aerr)
		}
		if _, err := gs.UpdateVariation(ctx, *sv); err != nil {
			t.Fatalf("UpdateVariation %d: %v", i, err)
		}
		approved = a
	}
	if !approved {
		t.Fatal("variation never reached approved after 4 advances")
	}
	if err := gs.AddContractValue(ctx, c.ID, sv.Amount); err != nil {
		t.Fatalf("AddContractValue: %v", err)
	}
	after, _ := gs.GetContract(ctx, c.ID)
	if after.Value != 105_000_000 {
		t.Fatalf("contract value after variation: got %d want 105000000", after.Value)
	}

	// --- Requisition ---
	r := models.NewRequisition(models.NewGovID("GREQ"), "REQ-1", "Buy rebar", 8_000_000, "pm", "now")
	if _, err := gs.CreateRequisition(ctx, r); err != nil {
		t.Fatalf("CreateRequisition: %v", err)
	}
	sr, _ := gs.GetRequisition(ctx, "REQ-1")
	if _, err := sr.Advance("dept", "now"); err != nil {
		t.Fatalf("requisition advance: %v", err)
	}
	if _, err := gs.UpdateRequisition(ctx, *sr); err != nil {
		t.Fatalf("UpdateRequisition: %v", err)
	}

	// --- Value-banded approval routing: narrowest active band wins ---
	band := int64(50_000_000)
	if _, err := gs.UpsertApprovalRule(ctx, models.GovApprovalRule{
		ID: models.NewGovID("GAR"), Name: "Low", MinValue: 0, MaxValue: &band,
		Route: []string{"PM", "Finance"}, Status: "active", // lowercase on purpose: asserts case-insensitive routing
	}); err != nil {
		t.Fatalf("UpsertApprovalRule low: %v", err)
	}
	if _, err := gs.UpsertApprovalRule(ctx, models.GovApprovalRule{
		ID: models.NewGovID("GAR"), Name: "High", MinValue: 50_000_000,
		Route: []string{"PM", "Finance", "Management", "Board"}, Status: "active", // lowercase on purpose: asserts case-insensitive routing
	}); err != nil {
		t.Fatalf("UpsertApprovalRule high: %v", err)
	}
	rules, err := gs.ListApprovalRules(ctx)
	if err != nil || len(rules) != 2 {
		t.Fatalf("ListApprovalRules: got %d err %v", len(rules), err)
	}
	route := models.ResolveApprovalRoute(rules, 10_000_000)
	if route == nil || route.Name != "Low" {
		t.Fatalf("resolve 10M: got %+v, want Low", route)
	}
	if hi := models.ResolveApprovalRoute(rules, 80_000_000); hi == nil || hi.Name != "High" {
		t.Fatalf("resolve 80M: got %+v, want High", hi)
	}
}

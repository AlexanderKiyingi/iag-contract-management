package models

import "fmt"

// seed establishes the starting shape of an unhydrated store.
//
// It used to build a demo workspace — five zones, 96 contracts, 32 milestones, 48 audit
// entries, 56 material lines and 8 task projects with invented supervisors and
// engineers. That data was removed from every environment by migration
// 018_purge_seeded_monthly_report.up.sql together with the seeded gov_* rows, and
// config no longer allows seeding in production at all.
//
// What remains is deliberately not business data: empty collections with their maps
// initialised, plus the two custom role definitions. Roles and their permission sets are
// authorisation configuration, not demo records, so they survive the purge — a fresh
// deployment still offers Zone Lead and Auditor to assign.
//
// NewStore reloads everything from Postgres immediately after calling this, so these
// values only ever surface when the service runs without a database.
func seed(s *Store) {
	s.Workspace = buildWorkspaceSeed()
	s.Frontend = buildFrontendSeed()
}

func buildWorkspaceSeed() Workspace {
	return Workspace{
		Zones:     []Zone{},
		Contracts: []Contract{},
	}
}

func buildFrontendSeed() FrontendStore {
	managerTpl := "manager"
	viewerTpl := "viewer"
	return FrontendStore{
		Tasks:      TasksStore{Projects: []TaskProject{}},
		Milestones: []Milestone{},
		Assistance: []AssistanceMessage{},
		Audit:      []AuditEntry{},
		Materials:  []MaterialEntry{},
		Updates:    []any{},
		CustomRoles: []CustomRole{
			{
				ID:          "cr_zone_lead",
				Name:        "Zone Lead",
				Description: "Field coordination — CRUD on contracts & operations",
				Permissions: []string{
					"contracts.create", "contracts.read", "contracts.update",
					"zones.read", "zones.update",
					"payments.read",
					"tasks.create", "tasks.read", "tasks.update",
					"milestones.read", "milestones.update",
				},
				Template: &managerTpl,
			},
			{
				ID:          "cr_auditor",
				Name:        "Auditor",
				Description: "Read-only across modules + report export",
				Permissions: append(readAllModulePerms(), "reports.create"),
				Template:    &viewerTpl,
			},
		},
		ProfilePhotos: map[string]string{},
	}
}

func readAllModulePerms() []string {
	modules := []string{
		"contracts", "zones", "payments", "tasks", "milestones", "materials",
		"users", "roles", "audit", "reports", "insights",
	}
	out := make([]string, 0, len(modules))
	for _, m := range modules {
		out = append(out, m+".read")
	}
	return out
}

func recalcZoneCounts(ws *Workspace) {
	counts := map[string]int{}
	for _, c := range ws.Contracts {
		counts[c.Zone]++
	}
	for i := range ws.Zones {
		ws.Zones[i].Contracts = counts[ws.Zones[i].Code]
	}
}

func nextContractNo(contracts []Contract) string {
	return fmt.Sprintf("C-%04d", len(contracts)+1)
}

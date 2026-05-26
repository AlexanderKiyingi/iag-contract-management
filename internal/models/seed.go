package models

import (
	"fmt"
	"strings"
)

var (
	zoneColors = []string{"#059669", "#d97706", "#2563eb", "#7c3aed", "#dc2626"}
	zoneNames  = []string{"North Ridge", "Valley Works", "Plateau East", "River Basin", "Summit South"}
	supervisors = []string{
		"Favour Kyorishaba", "James Okello", "Mary Nalwoga", "Peter Mugisha", "Grace Akello",
	}
	engNames = []string{
		"James Okello", "Mary Nalwoga", "Peter Mugisha", "Grace Akello", "David Ssemakula",
		"Ruth Namuli", "Samuel Kato", "Irene Nabwire", "Brian Tumwine", "Catherine Achieng",
		"Moses Wasswa", "Patricia Laker", "Henry Ochieng", "Faith Nakato", "Emmanuel Byaruhanga",
	}
	contractTypes = []string{
		"Roads", "Drainage", "Structures", "Utilities", "Landscaping", "Bridges", "Electrical", "Water Supply",
	}
	materialItems = []string{
		"Cement 50kg", "Steel bars 12mm", "Aggregate", "Timber truss", "PVC pipe",
		"Rebar 16mm", "Sand fine", "Gravel", "Roofing sheet", "Binding wire",
	}
	statuses = []ContractStatus{StatusPlanning, StatusActive, StatusOnHold, StatusComplete}
	priorities = []Priority{PriorityHigh, PriorityMedium, PriorityLow}
)

func seed(s *Store) {
	s.Workspace = buildWorkspaceSeed()
	s.Frontend = buildFrontendSeed()
}

func buildWorkspaceSeed() Workspace {
	zones := make([]Zone, len(zoneNames))
	for i, name := range zoneNames {
		cs := int64(800000000 + i*120000000)
		paid := int64(float64(cs) * (0.35 + float64(i)*0.08))
		zones[i] = Zone{
			Code:  fmt.Sprintf("Z%d", i+1),
			Name:  name,
			Desc:  fmt.Sprintf("Construction zone — %s", name),
			Sup:   supervisors[i%5],
			Cs:    cs,
			Paid:  paid,
			Bal:   cs - paid,
			Color: zoneColors[i],
		}
	}

	contracts := make([]Contract, 0, 96)
	for i := 1; i <= 96; i++ {
		zone := zones[(i-1)%5]
		cs := int64(38000000 + ((i * 991000) % 95000000))
		paid := int64(float64(cs) * (0.18 + float64(i%9)*0.08))
		month := fmt.Sprintf("%02d", 1+(i%5))
		day := fmt.Sprintf("%02d", 1+(i%28))
		remarks := ""
		if i%7 == 0 {
			remarks = "At risk"
		} else if i%11 == 0 {
			remarks = "Review due"
		}
		contracts = append(contracts, Contract{
			No:      fmt.Sprintf("C-%04d", i),
			Name:    fmt.Sprintf("Contract %d — %s", i, contractTypes[(i-1)%len(contractTypes)]),
			Zone:    zone.Code,
			Cs:      cs,
			Paid:    paid,
			Bal:     cs - paid,
			Prog:    12 + ((i * 13) % 88),
			Status:  statuses[i%4],
			Pri:     priorities[i%3],
			Workers: 6 + (i % 24),
			Sup:     supervisors[i%5],
			Remarks: remarks,
			Created: fmt.Sprintf("2026-%s-%s", month, day),
		})
	}

	for i := range zones {
		count := 0
		for _, c := range contracts {
			if c.Zone == zones[i].Code {
				count++
			}
		}
		zones[i].Contracts = count
	}

	engineers := make([]Engineer, len(engNames))
	for i, name := range engNames {
		role := "Resident Engineer"
		if i < 8 {
			role = "Site Engineer"
		} else if i < 20 {
			role = "Supervisor"
		}
		active := "Active"
		if i%9 == 0 {
			active = "On leave"
		}
		email := strings.ToLower(strings.ReplaceAll(name, " ", ".")) + "@inspireafrica.test"
		engineers[i] = Engineer{
			ID:     fmt.Sprintf("ENG-%02d", i+1),
			Name:   name,
			Role:   role,
			Zone:   zones[i%5].Code,
			Phone:  fmt.Sprintf("+256 7%09d", (700000000+i*1234567)%1000000000),
			Email:  email,
			Active: active,
		}
	}

	roles := []string{"super_admin", "admin", "manager", "manager", "viewer", "viewer", "manager", "admin"}
	users := []WorkspaceUser{
		{ID: "u1", Email: "admin@inspireafrica.test", DisplayName: "System Admin", Role: "super_admin", Status: "active"},
		{ID: "u2", Email: "ops@inspireafrica.test", DisplayName: "Ops Manager", Role: "manager", Status: "active"},
		{ID: "u3", Email: "view@inspireafrica.test", DisplayName: "Portfolio Viewer", Role: "viewer", Status: "active"},
	}
	for i := 0; i < 12; i++ {
		status := "active"
		if i%5 == 0 {
			status = "inactive"
		}
		users = append(users, WorkspaceUser{
			ID:          fmt.Sprintf("u%d", i+4),
			Email:       fmt.Sprintf("user%d@inspireafrica.test", i+4),
			DisplayName: fmt.Sprintf("Staff Member %d", i+4),
			Role:        roles[i%len(roles)],
			Status:      status,
		})
	}

	contractors := make([]Contractor, len(supervisors))
	for i, name := range supervisors {
		contractors[i] = Contractor{ID: fmt.Sprintf("ct%d", i+1), Name: name}
	}

	return Workspace{
		Zones:          zones,
		Contracts:      contracts,
		Engineers:      engineers,
		WorkspaceUsers: users,
		Contractors:    contractors,
	}
}

func buildFrontendSeed() FrontendStore {
	milestones := make([]Milestone, 32)
	milestoneKinds := []string{"Foundation", "Structural", "MEP", "Finishing", "Handover"}
	milestoneStatuses := []string{"Pending", "In progress", "Complete"}
	for i := 0; i < 32; i++ {
		milestones[i] = Milestone{
			ID:     fmt.Sprintf("M-%d", i+1),
			Title:  fmt.Sprintf("Milestone %d: %s", i+1, milestoneKinds[i%5]),
			Due:    fmt.Sprintf("2026-%02d-%02d", 3+(i%9), 5+(i%20)),
			Zone:   fmt.Sprintf("Z%d", (i%5)+1),
			Status: milestoneStatuses[i%3],
			Owner:  supervisors[i%5],
		}
	}

	auditActions := []string{"Contract updated", "Zone saved", "User added", "Export CSV", "Login", "Status change"}
	audit := make([]AuditEntry, 48)
	for i := 0; i < 48; i++ {
		user := "Ops Manager"
		if i%2 == 0 {
			user = "System Admin"
		}
		audit[i] = AuditEntry{
			ID:     fmt.Sprintf("AUD-%d", 1000+i),
			At:     fmt.Sprintf("2026-05-%02d %d:%02d", 1+(i%20), 8+(i%12), i%60),
			User:   user,
			Action: auditActions[i%6],
			Detail: fmt.Sprintf("Record C-%04d — batch %d", 1+(i%96), i+1),
		}
	}

	materials := make([]MaterialEntry, 56)
	for i := 0; i < 56; i++ {
		unit := "tonnes"
		if i%2 != 0 {
			unit = "bags"
		}
		materials[i] = MaterialEntry{
			ID:       fmt.Sprintf("MAT-%d", i+1),
			Item:     materialItems[i%len(materialItems)],
			Zone:     fmt.Sprintf("Z%d", (i%5)+1),
			Qty:      120 + ((i * 17) % 900),
			Unit:     unit,
			Date:     fmt.Sprintf("2026-04-%02d", 1+(i%28)),
			Supplier: fmt.Sprintf("Supplier %d", 1+(i%8)),
		}
	}

	projects := make([]TaskProject, 8)
	for p := 0; p < 8; p++ {
		tasks := make([]TaskItem, 6)
		for t := 0; t < 6; t++ {
			cols := []string{"Backlog", "In progress", "Done"}
			tasks[t] = TaskItem{
				ID:       fmt.Sprintf("T-%d-%d", p, t),
				Title:    fmt.Sprintf("Task %d", t+1),
				Col:      cols[t%3],
				Assignee: engNames[(p+t)%len(engNames)],
			}
		}
		projects[p] = TaskProject{
			Name:     fmt.Sprintf("Project %d — %s", p+1, zoneNames[p%5]),
			Sections: []string{"Planning", "Execution", "Close-out"},
			Tasks:    tasks,
		}
	}

	managerTpl := "manager"
	viewerTpl := "viewer"
	return FrontendStore{
		Tasks:      TasksStore{Projects: projects},
		Milestones: milestones,
		Assistance: []AssistanceMessage{},
		Audit:      audit,
		Materials:  materials,
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

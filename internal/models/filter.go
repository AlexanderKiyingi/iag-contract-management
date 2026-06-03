package models

// FilterWorkspace returns a copy of ws containing only the entities the
// supplied session is permitted to see. Slices the caller can't read collapse
// to empty slices (NOT nil — keeps the JSON shape stable for the UI).
// Contractor scope further trims contracts to those supervised by the
// caller's ContractorSup mapping.
func FilterWorkspace(ws Workspace, sess Session) Workspace {
	out := Workspace{
		Zones:          []Zone{},
		Contracts:      []Contract{},
		Engineers:      []Engineer{},
		WorkspaceUsers: []WorkspaceUser{},
		Contractors:    []Contractor{},
	}

	if sessionHasPermission(sess, "zones.read") {
		out.Zones = append([]Zone(nil), ws.Zones...)
	}
	if sessionHasPermission(sess, "contracts.read") {
		out.Contracts = filterContractsForSession(ws.Contracts, sess)
	}
	if sessionHasPermission(sess, "users.read") {
		out.Engineers = append([]Engineer(nil), ws.Engineers...)
		out.WorkspaceUsers = append([]WorkspaceUser(nil), ws.WorkspaceUsers...)
		out.Contractors = append([]Contractor(nil), ws.Contractors...)
	}
	return out
}

// FilterFrontend returns a copy of fe containing only the entities the
// supplied session is permitted to see.
func FilterFrontend(fe FrontendStore, sess Session) FrontendStore {
	out := FrontendStore{
		Tasks:         TasksStore{Projects: []TaskProject{}},
		Milestones:    []Milestone{},
		Assistance:    []AssistanceMessage{},
		Audit:         []AuditEntry{},
		Materials:     []MaterialEntry{},
		Updates:       []any{},
		CustomRoles:   []CustomRole{},
		ProfilePhotos: map[string]string{},
		// AiScan retained when caller can read insights; nil otherwise.
	}

	if sessionHasPermission(sess, "tasks.read") {
		out.Tasks = TasksStore{Projects: append([]TaskProject(nil), fe.Tasks.Projects...)}
	}
	if sessionHasPermission(sess, "milestones.read") {
		out.Milestones = append([]Milestone(nil), fe.Milestones...)
	}
	if sessionHasPermission(sess, "materials.read") {
		out.Materials = append([]MaterialEntry(nil), fe.Materials...)
	}
	if sessionHasPermission(sess, "audit.read") {
		out.Audit = append([]AuditEntry(nil), fe.Audit...)
	}
	if sessionHasPermission(sess, "roles.read") {
		out.CustomRoles = append([]CustomRole(nil), fe.CustomRoles...)
	}
	if sessionHasPermission(sess, "insights.read") {
		out.AiScan = fe.AiScan
		out.Updates = append([]any(nil), fe.Updates...)
	}

	// Assistance: staff see all threads; others see only their own messages.
	if sessionCanMutate(sess) {
		out.Assistance = append([]AssistanceMessage(nil), fe.Assistance...)
	} else if sess.Email != "" {
		for _, m := range fe.Assistance {
			if m.From == sess.Email {
				out.Assistance = append(out.Assistance, m)
			}
		}
	}
	if sess.Email != "" {
		if photo, ok := fe.ProfilePhotos[sess.Email]; ok {
			out.ProfilePhotos[sess.Email] = photo
		}
	}
	// Staff/admin/manager can see ALL profile photos (used to render team views).
	if sessionCanMutate(sess) {
		for k, v := range fe.ProfilePhotos {
			out.ProfilePhotos[k] = v
		}
	}
	return out
}

// filterContractsForSession scopes contracts to those a contractor's
// supervisor mapping permits. Non-contractor sessions see everything they
// have contracts.read for.
func filterContractsForSession(contracts []Contract, sess Session) []Contract {
	if sess.Role != "contractor" || sess.ContractorSup == nil {
		return append([]Contract(nil), contracts...)
	}
	sup := *sess.ContractorSup
	out := make([]Contract, 0, len(contracts))
	for _, c := range contracts {
		if c.Sup == sup {
			out = append(out, c)
		}
	}
	return out
}

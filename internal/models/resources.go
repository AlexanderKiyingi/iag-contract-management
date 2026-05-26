package models

import (
	"fmt"
	"strings"
	"time"
)

func (s *Store) ListZones() []Zone {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Zone, len(s.Workspace.Zones))
	copy(out, s.Workspace.Zones)
	return out
}

func (s *Store) GetZone(code string) (_ Zone, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, z := range s.Workspace.Zones {
		if z.Code == code {
			return z, nil
		}
	}
	return Zone{}, ErrNotFound
}

func (s *Store) GetEngineer(id string) (_ Engineer, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.Workspace.Engineers {
		if e.ID == id {
			return e, nil
		}
	}
	return Engineer{}, ErrNotFound
}

func (s *Store) ListEngineers() []Engineer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Engineer, len(s.Workspace.Engineers))
	copy(out, s.Workspace.Engineers)
	return out
}

func (s *Store) CreateEngineer(in EngineerInput) (Engineer, error) {
	if strings.TrimSpace(in.Name) == "" {
		return Engineer{}, ErrValidation
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	eng := Engineer{
		ID:     fmt.Sprintf("ENG-%03d", len(s.Workspace.Engineers)+1),
		Name:   strings.TrimSpace(in.Name),
		Role:   strings.TrimSpace(in.Role),
		Zone:   in.Zone,
		Phone:  strings.TrimSpace(in.Phone),
		Email:  strings.ToLower(strings.TrimSpace(in.Email)),
		Active: in.Active,
	}
	if s.hasRepo() {
		if err := s.repo.InsertEngineer(s.persistCtx(), eng); err != nil {
			return Engineer{}, wrapPersist(err)
		}
	}
	s.Workspace.Engineers = append(s.Workspace.Engineers, eng)
	return eng, nil
}

func (s *Store) PatchEngineer(id string, in EngineerInput) (Engineer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Workspace.Engineers {
		if s.Workspace.Engineers[i].ID != id {
			continue
		}
		updated := s.Workspace.Engineers[i]
		if strings.TrimSpace(in.Name) != "" {
			updated.Name = strings.TrimSpace(in.Name)
		}
		if in.Role != "" {
			updated.Role = strings.TrimSpace(in.Role)
		}
		if in.Zone != "" {
			updated.Zone = in.Zone
		}
		if in.Phone != "" {
			updated.Phone = strings.TrimSpace(in.Phone)
		}
		if in.Email != "" {
			updated.Email = strings.ToLower(strings.TrimSpace(in.Email))
		}
		if in.Active != "" {
			updated.Active = in.Active
		}
		if s.hasRepo() {
			if err := s.repo.UpdateEngineer(s.persistCtx(), updated); err != nil {
				return Engineer{}, wrapPersist(err)
			}
		}
		s.Workspace.Engineers[i] = updated
		return updated, nil
	}
	return Engineer{}, ErrNotFound
}

func (s *Store) DeleteEngineer(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.Workspace.Engineers {
		if e.ID != id {
			continue
		}
		if s.hasRepo() {
			if err := s.repo.DeleteEngineer(s.persistCtx(), id); err != nil {
				return wrapPersist(err)
			}
		}
		s.Workspace.Engineers = append(s.Workspace.Engineers[:i], s.Workspace.Engineers[i+1:]...)
		return nil
	}
	return ErrNotFound
}

func (s *Store) GetUser(id string) (_ WorkspaceUser, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.Workspace.WorkspaceUsers {
		if u.ID == id {
			return u, nil
		}
	}
	return WorkspaceUser{}, ErrNotFound
}

func (s *Store) ListUsers() []WorkspaceUser {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]WorkspaceUser, len(s.Workspace.WorkspaceUsers))
	copy(out, s.Workspace.WorkspaceUsers)
	return out
}

func (s *Store) CreateUser(in UserInput) (WorkspaceUser, error) {
	if strings.TrimSpace(in.DisplayName) == "" || strings.TrimSpace(in.Email) == "" {
		return WorkspaceUser{}, ErrValidation
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.Workspace.WorkspaceUsers {
		if strings.EqualFold(u.Email, email) {
			return WorkspaceUser{}, ErrConflict
		}
	}
	u := WorkspaceUser{
		ID:           fmt.Sprintf("USR-%03d", len(s.Workspace.WorkspaceUsers)+1),
		Email:        email,
		DisplayName:  strings.TrimSpace(in.DisplayName),
		Role:         in.Role,
		Status:       in.Status,
		CustomRoleID: in.CustomRoleID,
	}
	if s.hasRepo() {
		if err := s.repo.InsertWorkspaceUser(s.persistCtx(), u); err != nil {
			return WorkspaceUser{}, wrapPersist(err)
		}
	}
	s.Workspace.WorkspaceUsers = append(s.Workspace.WorkspaceUsers, u)
	return u, nil
}

func (s *Store) PatchUser(id string, patch UserPatch) (WorkspaceUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Workspace.WorkspaceUsers {
		if s.Workspace.WorkspaceUsers[i].ID != id {
			continue
		}
		updated := s.Workspace.WorkspaceUsers[i]
		if patch.DisplayName != nil {
			updated.DisplayName = strings.TrimSpace(*patch.DisplayName)
		}
		if patch.Email != nil {
			updated.Email = strings.ToLower(strings.TrimSpace(*patch.Email))
		}
		if patch.Role != nil {
			updated.Role = *patch.Role
		}
		if patch.Status != nil {
			updated.Status = *patch.Status
		}
		if patch.CustomRoleID != nil {
			updated.CustomRoleID = patch.CustomRoleID
		}
		if s.hasRepo() {
			if err := s.repo.UpdateWorkspaceUser(s.persistCtx(), updated); err != nil {
				return WorkspaceUser{}, wrapPersist(err)
			}
		}
		s.Workspace.WorkspaceUsers[i] = updated
		return updated, nil
	}
	return WorkspaceUser{}, ErrNotFound
}

func (s *Store) DeactivateUser(id string) error {
	status := "inactive"
	_, err := s.PatchUser(id, UserPatch{Status: &status})
	return err
}

func (s *Store) ListMilestones() []Milestone {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Milestone, len(s.Frontend.Milestones))
	copy(out, s.Frontend.Milestones)
	return out
}

func (s *Store) GetMilestone(id string) (_ Milestone, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, m := range s.Frontend.Milestones {
		if m.ID == id {
			return m, nil
		}
	}
	return Milestone{}, ErrNotFound
}

func (s *Store) CreateMilestone(in MilestoneInput) (Milestone, error) {
	if strings.TrimSpace(in.Title) == "" {
		return Milestone{}, ErrValidation
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m := Milestone{
		ID:     fmt.Sprintf("MIL-%03d", len(s.Frontend.Milestones)+1),
		Title:  strings.TrimSpace(in.Title),
		Zone:   in.Zone,
		Due:    in.Due,
		Owner:  in.Owner,
		Status: in.Status,
	}
	if s.hasRepo() {
		if err := s.repo.InsertMilestone(s.persistCtx(), m); err != nil {
			return Milestone{}, wrapPersist(err)
		}
	}
	s.Frontend.Milestones = append(s.Frontend.Milestones, m)
	return m, nil
}

func (s *Store) PatchMilestone(id string, patch MilestonePatch) (Milestone, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Frontend.Milestones {
		if s.Frontend.Milestones[i].ID != id {
			continue
		}
		updated := s.Frontend.Milestones[i]
		if patch.Title != nil {
			updated.Title = *patch.Title
		}
		if patch.Zone != nil {
			updated.Zone = *patch.Zone
		}
		if patch.Due != nil {
			updated.Due = *patch.Due
		}
		if patch.Owner != nil {
			updated.Owner = *patch.Owner
		}
		if patch.Status != nil {
			updated.Status = *patch.Status
		}
		if s.hasRepo() {
			if err := s.repo.UpdateMilestone(s.persistCtx(), updated); err != nil {
				return Milestone{}, wrapPersist(err)
			}
		}
		s.Frontend.Milestones[i] = updated
		return updated, nil
	}
	return Milestone{}, ErrNotFound
}

func (s *Store) DeleteMilestone(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, m := range s.Frontend.Milestones {
		if m.ID != id {
			continue
		}
		if s.hasRepo() {
			if err := s.repo.DeleteMilestone(s.persistCtx(), id); err != nil {
				return wrapPersist(err)
			}
		}
		s.Frontend.Milestones = append(s.Frontend.Milestones[:i], s.Frontend.Milestones[i+1:]...)
		return nil
	}
	return ErrNotFound
}

func (s *Store) ListMaterials() []MaterialEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MaterialEntry, len(s.Frontend.Materials))
	copy(out, s.Frontend.Materials)
	return out
}

func (s *Store) CreateMaterial(in MaterialInput) (MaterialEntry, error) {
	if strings.TrimSpace(in.Item) == "" {
		return MaterialEntry{}, ErrValidation
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m := MaterialEntry{
		ID:       fmt.Sprintf("MAT-%03d", len(s.Frontend.Materials)+1),
		Item:     in.Item,
		Zone:     in.Zone,
		Qty:      in.Qty,
		Unit:     in.Unit,
		Supplier: in.Supplier,
		Date:     in.Date,
	}
	if s.hasRepo() {
		if err := s.repo.InsertMaterial(s.persistCtx(), m); err != nil {
			return MaterialEntry{}, wrapPersist(err)
		}
	}
	s.Frontend.Materials = append(s.Frontend.Materials, m)
	return m, nil
}

func (s *Store) PatchMaterial(id string, patch MaterialPatch) (MaterialEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Frontend.Materials {
		if s.Frontend.Materials[i].ID != id {
			continue
		}
		updated := s.Frontend.Materials[i]
		if patch.Item != nil {
			updated.Item = *patch.Item
		}
		if patch.Zone != nil {
			updated.Zone = *patch.Zone
		}
		if patch.Qty != nil {
			updated.Qty = *patch.Qty
		}
		if patch.Unit != nil {
			updated.Unit = *patch.Unit
		}
		if patch.Supplier != nil {
			updated.Supplier = *patch.Supplier
		}
		if patch.Date != nil {
			updated.Date = *patch.Date
		}
		if s.hasRepo() {
			if err := s.repo.UpdateMaterial(s.persistCtx(), updated); err != nil {
				return MaterialEntry{}, wrapPersist(err)
			}
		}
		s.Frontend.Materials[i] = updated
		return updated, nil
	}
	return MaterialEntry{}, ErrNotFound
}

func (s *Store) DeleteMaterial(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, m := range s.Frontend.Materials {
		if m.ID != id {
			continue
		}
		if s.hasRepo() {
			if err := s.repo.DeleteMaterial(s.persistCtx(), id); err != nil {
				return wrapPersist(err)
			}
		}
		s.Frontend.Materials = append(s.Frontend.Materials[:i], s.Frontend.Materials[i+1:]...)
		return nil
	}
	return ErrNotFound
}

func (s *Store) ListProjects() []TaskProject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Frontend.Tasks.Projects == nil {
		return []TaskProject{}
	}
	out := make([]TaskProject, len(s.Frontend.Tasks.Projects))
	copy(out, s.Frontend.Tasks.Projects)
	return out
}

func (s *Store) CreateProject(in ProjectInput) (TaskProject, int, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = "New project"
	}
	sections := []string{"Planning", "Execution", "Close-out"}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := TaskProject{
		Name:     name,
		Sections: sections,
		Tasks:    []TaskItem{},
	}
	if s.hasRepo() {
		id, err := s.repo.InsertTaskProject(s.persistCtx(), name, sections)
		if err != nil {
			return TaskProject{}, 0, wrapPersist(err)
		}
		p.DBID = id
	}
	s.Frontend.Tasks.Projects = append(s.Frontend.Tasks.Projects, p)
	return p, len(s.Frontend.Tasks.Projects) - 1, nil
}

func (s *Store) PatchProject(index int, patch ProjectPatch) (TaskProject, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.Frontend.Tasks.Projects) {
		return TaskProject{}, ErrNotFound
	}
	updated := s.Frontend.Tasks.Projects[index]
	if patch.Name != nil {
		updated.Name = strings.TrimSpace(*patch.Name)
	}
	if s.hasRepo() && updated.DBID != 0 {
		if err := s.repo.UpdateTaskProject(s.persistCtx(), updated.DBID, updated.Name, updated.Sections); err != nil {
			return TaskProject{}, wrapPersist(err)
		}
	}
	s.Frontend.Tasks.Projects[index] = updated
	return updated, nil
}

func (s *Store) DeleteProject(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.Frontend.Tasks.Projects) {
		return ErrNotFound
	}
	proj := s.Frontend.Tasks.Projects[index]
	if s.hasRepo() && proj.DBID != 0 {
		if err := s.repo.DeleteTaskProject(s.persistCtx(), proj.DBID); err != nil {
			return wrapPersist(err)
		}
	}
	s.Frontend.Tasks.Projects = append(s.Frontend.Tasks.Projects[:index], s.Frontend.Tasks.Projects[index+1:]...)
	return nil
}

func (s *Store) CreateTask(projectIndex int, in TaskInput) (TaskItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if projectIndex < 0 || projectIndex >= len(s.Frontend.Tasks.Projects) {
		return TaskItem{}, ErrNotFound
	}
	proj := &s.Frontend.Tasks.Projects[projectIndex]
	t := TaskItem{
		ID:       fmt.Sprintf("T-%d", time.Now().UnixMilli()),
		Title:    strings.TrimSpace(in.Title),
		Col:      in.Col,
		Assignee: in.Assignee,
	}
	if s.hasRepo() && proj.DBID != 0 {
		if err := s.repo.InsertTaskItem(s.persistCtx(), proj.DBID, t); err != nil {
			return TaskItem{}, wrapPersist(err)
		}
	}
	proj.Tasks = append(proj.Tasks, t)
	return t, nil
}

func (s *Store) PatchTask(projectIndex int, taskID string, patch TaskPatch) (TaskItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if projectIndex < 0 || projectIndex >= len(s.Frontend.Tasks.Projects) {
		return TaskItem{}, ErrNotFound
	}
	proj := &s.Frontend.Tasks.Projects[projectIndex]
	for i := range proj.Tasks {
		if proj.Tasks[i].ID != taskID {
			continue
		}
		updated := proj.Tasks[i]
		if patch.Title != nil {
			updated.Title = *patch.Title
		}
		if patch.Col != nil {
			updated.Col = *patch.Col
		}
		if patch.Assignee != nil {
			updated.Assignee = *patch.Assignee
		}
		if s.hasRepo() && proj.DBID != 0 {
			if err := s.repo.UpdateTaskItem(s.persistCtx(), proj.DBID, updated); err != nil {
				return TaskItem{}, wrapPersist(err)
			}
		}
		proj.Tasks[i] = updated
		return updated, nil
	}
	return TaskItem{}, ErrNotFound
}

func (s *Store) DeleteTask(projectIndex int, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if projectIndex < 0 || projectIndex >= len(s.Frontend.Tasks.Projects) {
		return ErrNotFound
	}
	proj := &s.Frontend.Tasks.Projects[projectIndex]
	for i, t := range proj.Tasks {
		if t.ID != taskID {
			continue
		}
		if s.hasRepo() && proj.DBID != 0 {
			if err := s.repo.DeleteTaskItem(s.persistCtx(), proj.DBID, taskID); err != nil {
				return wrapPersist(err)
			}
		}
		proj.Tasks = append(proj.Tasks[:i], proj.Tasks[i+1:]...)
		return nil
	}
	return ErrNotFound
}

func (s *Store) ListRoles() []CustomRole {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]CustomRole, len(s.Frontend.CustomRoles))
	copy(out, s.Frontend.CustomRoles)
	return out
}

func (s *Store) GetRole(id string) (_ CustomRole, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.Frontend.CustomRoles {
		if r.ID == id {
			return r, nil
		}
	}
	return CustomRole{}, ErrNotFound
}

func (s *Store) CreateRole(in RoleInput) (CustomRole, error) {
	if strings.TrimSpace(in.Name) == "" {
		return CustomRole{}, ErrValidation
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := CustomRole{
		ID:          fmt.Sprintf("ROLE-%03d", len(s.Frontend.CustomRoles)+1),
		Name:        strings.TrimSpace(in.Name),
		Description: strings.TrimSpace(in.Description),
		Permissions: NormalizePermissions(in.Permissions),
		Template:    in.Template,
	}
	if s.hasRepo() {
		if err := s.repo.InsertCustomRole(s.persistCtx(), r); err != nil {
			return CustomRole{}, wrapPersist(err)
		}
	}
	s.Frontend.CustomRoles = append(s.Frontend.CustomRoles, r)
	return r, nil
}

func (s *Store) PatchRole(id string, patch RolePatch) (CustomRole, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Frontend.CustomRoles {
		if s.Frontend.CustomRoles[i].ID != id {
			continue
		}
		updated := s.Frontend.CustomRoles[i]
		if patch.Name != nil {
			updated.Name = *patch.Name
		}
		if patch.Description != nil {
			updated.Description = *patch.Description
		}
		if patch.Permissions != nil {
			updated.Permissions = NormalizePermissions(patch.Permissions)
		}
		if patch.Template != nil {
			updated.Template = patch.Template
		}
		if s.hasRepo() {
			if err := s.repo.UpdateCustomRole(s.persistCtx(), updated); err != nil {
				return CustomRole{}, wrapPersist(err)
			}
		}
		s.Frontend.CustomRoles[i] = updated
		return updated, nil
	}
	return CustomRole{}, ErrNotFound
}

func (s *Store) DeleteRole(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.Frontend.CustomRoles {
		if r.ID != id {
			continue
		}
		if s.hasRepo() {
			if err := s.repo.DeleteCustomRole(s.persistCtx(), id); err != nil {
				return wrapPersist(err)
			}
		}
		s.Frontend.CustomRoles = append(s.Frontend.CustomRoles[:i], s.Frontend.CustomRoles[i+1:]...)
		// DB cascade already cleared the FK; mirror the same in memory.
		for j := range s.Workspace.WorkspaceUsers {
			if s.Workspace.WorkspaceUsers[j].CustomRoleID != nil && *s.Workspace.WorkspaceUsers[j].CustomRoleID == id {
				s.Workspace.WorkspaceUsers[j].CustomRoleID = nil
			}
		}
		return nil
	}
	return ErrNotFound
}

func (s *Store) ListAudit() []AuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AuditEntry, len(s.Frontend.Audit))
	copy(out, s.Frontend.Audit)
	return out
}

func (s *Store) GetAudit(id string) (_ AuditEntry, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.Frontend.Audit {
		if a.ID == id {
			return a, nil
		}
	}
	return AuditEntry{}, ErrNotFound
}

// AppendAudit appends an audit entry attributed to user. The caller (the
// audit controller) is expected to pass the session display name from the
// request context — pre-cutover this came from an in-memory default session.
func (s *Store) AppendAudit(in AuditInput, user string) (AuditEntry, error) {
	if strings.TrimSpace(in.Action) == "" {
		return AuditEntry{}, ErrValidation
	}
	if strings.TrimSpace(user) == "" {
		user = "unknown"
	}
	s.appendAudit(in.Action, in.Detail, user)
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.Frontend.Audit) == 0 {
		return AuditEntry{}, nil
	}
	return s.Frontend.Audit[0], nil
}

func (s *Store) ListAssistance() []AssistanceMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AssistanceMessage, len(s.Frontend.Assistance))
	copy(out, s.Frontend.Assistance)
	return out
}

// PostAssistance records an assistance message attributed to from.
func (s *Store) PostAssistance(text, from string) (AssistanceMessage, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return AssistanceMessage{}, ErrValidation
	}
	if strings.TrimSpace(from) == "" {
		from = "unknown"
	}
	msg := AssistanceMessage{
		From: from,
		Text: text,
		At:   time.Now().Format("2006-01-02 15:04"),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hasRepo() {
		if err := s.repo.InsertAssistance(s.persistCtx(), msg); err != nil {
			return AssistanceMessage{}, wrapPersist(err)
		}
	}
	s.Frontend.Assistance = append([]AssistanceMessage{msg}, s.Frontend.Assistance...)
	return msg, nil
}

func (s *Store) SetProfilePhoto(email, dataURL string) error {
	if err := s.ValidateProfilePhotoDataURL(dataURL); err != nil {
		return err
	}
	key := strings.ToLower(strings.TrimSpace(email))
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Frontend.ProfilePhotos == nil {
		s.Frontend.ProfilePhotos = map[string]string{}
	}
	if dataURL == "" {
		if s.hasRepo() {
			if err := s.repo.DeleteProfilePhoto(s.persistCtx(), key); err != nil {
				return wrapPersist(err)
			}
		}
		delete(s.Frontend.ProfilePhotos, key)
		return nil
	}
	if s.hasRepo() {
		if err := s.repo.UpsertProfilePhoto(s.persistCtx(), key, dataURL); err != nil {
			return wrapPersist(err)
		}
	}
	s.Frontend.ProfilePhotos[key] = dataURL
	return nil
}

func (s *Store) SetAiScan(scan any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hasRepo() {
		if err := s.repo.UpsertAiScan(s.persistCtx(), scan); err != nil {
			logPersistError(err)
			return
		}
	}
	s.Frontend.AiScan = scan
}

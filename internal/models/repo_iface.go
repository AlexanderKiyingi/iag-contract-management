package models

import "context"

// SnapshotRepository persists workspace and frontend snapshots to PostgreSQL.
// Used by the super_admin-only bulk PUT endpoints and the initial seed path.
type SnapshotRepository interface {
	LoadState(ctx context.Context) (Workspace, FrontendStore, error)
	SaveState(ctx context.Context, ws Workspace, fe FrontendStore) error
	IsEmpty(ctx context.Context) (bool, error)
}

// EntityRepository writes individual rows for the per-entity REST endpoints.
// Replaces the previous "rewrite every table on each mutation" approach so
// concurrent writers don't trample each other.
type EntityRepository interface {
	// Contracts
	InsertContract(ctx context.Context, c Contract) error
	UpdateContract(ctx context.Context, c Contract) error
	DeleteContract(ctx context.Context, no string) error

	// Zones (only the derived contract_count is mutated by per-row writes;
	// the zone seed itself comes from the snapshot path).
	UpdateZone(ctx context.Context, z Zone) error
	UpdateZoneCount(ctx context.Context, code string, count int) error

	// Engineers
	InsertEngineer(ctx context.Context, e Engineer) error
	UpdateEngineer(ctx context.Context, e Engineer) error
	DeleteEngineer(ctx context.Context, id string) error

	// Workspace users
	InsertWorkspaceUser(ctx context.Context, u WorkspaceUser) error
	UpdateWorkspaceUser(ctx context.Context, u WorkspaceUser) error
	DeleteWorkspaceUser(ctx context.Context, id string) error

	// Milestones
	InsertMilestone(ctx context.Context, m Milestone) error
	UpdateMilestone(ctx context.Context, m Milestone) error
	DeleteMilestone(ctx context.Context, id string) error

	// Materials
	InsertMaterial(ctx context.Context, m MaterialEntry) error
	UpdateMaterial(ctx context.Context, m MaterialEntry) error
	DeleteMaterial(ctx context.Context, id string) error

	// Custom roles. DeleteCustomRole must also null out workspace_users
	// pointing at the role atomically.
	InsertCustomRole(ctx context.Context, r CustomRole) error
	UpdateCustomRole(ctx context.Context, r CustomRole) error
	DeleteCustomRole(ctx context.Context, id string) error

	// Audit log. InsertAuditAndTrim inserts a row and prunes to `keep` most-
	// recent entries in the same transaction.
	InsertAuditAndTrim(ctx context.Context, a AuditEntry, keep int) error

	// Assistance messages (append-only).
	InsertAssistance(ctx context.Context, m AssistanceMessage) error

	// Profile photos
	UpsertProfilePhoto(ctx context.Context, email, dataURL string) error
	DeleteProfilePhoto(ctx context.Context, email string) error

	// AI scan blob
	UpsertAiScan(ctx context.Context, scan any) error

	// Task projects + items. InsertTaskProject returns the surrogate id so
	// callers can attach subsequent task_items to the right project.
	InsertTaskProject(ctx context.Context, name string, sections []string) (int, error)
	UpdateTaskProject(ctx context.Context, id int, name string, sections []string) error
	DeleteTaskProject(ctx context.Context, id int) error
	InsertTaskItem(ctx context.Context, projectID int, t TaskItem) error
	UpdateTaskItem(ctx context.Context, projectID int, t TaskItem) error
	DeleteTaskItem(ctx context.Context, projectID int, taskID string) error
}

// Repository is the combined surface the store uses for persistence.
type Repository interface {
	SnapshotRepository
	EntityRepository
}

// ContractorMap resolves email → supervisor for the contractor portal scope.
// The auth service holds the user identity; this service holds the
// per-deployment mapping between a contractor email and the supervisor whose
// contracts they may edit.
type ContractorMap interface {
	ContractorSupervisor(ctx context.Context, email string) (string, bool, error)
	UpsertContractor(ctx context.Context, email, supervisor string) error
	RemoveContractor(ctx context.Context, email string) error
}

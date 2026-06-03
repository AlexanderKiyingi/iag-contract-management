package models

type ContractStatus string
type Priority string

const (
	StatusPlanning ContractStatus = "Planning"
	StatusActive   ContractStatus = "Active"
	StatusOnHold   ContractStatus = "On Hold"
	StatusComplete ContractStatus = "Complete"

	PriorityHigh   Priority = "High"
	PriorityMedium Priority = "Medium"
	PriorityLow    Priority = "Low"
)

type Contract struct {
	No      string         `json:"no"`
	Name    string         `json:"name"`
	Zone    string         `json:"zone"`
	Cs      int64          `json:"cs"`
	Paid    int64          `json:"paid"`
	Bal     int64          `json:"bal"`
	Prog    int            `json:"prog"`
	Status  ContractStatus `json:"status"`
	Pri     Priority       `json:"pri"`
	Workers int            `json:"workers"`
	Sup     string         `json:"sup"`
	Remarks string         `json:"remarks"`
	Created string         `json:"created"`
}

type Zone struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Desc      string `json:"desc"`
	Sup       string `json:"sup"`
	Cs        int64  `json:"cs"`
	Paid      int64  `json:"paid"`
	Bal       int64  `json:"bal"`
	Color     string `json:"color"`
	Contracts int    `json:"contracts"`
}

type Engineer struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Zone   string `json:"zone"`
	Phone  string `json:"phone"`
	Email  string `json:"email"`
	Active string `json:"active"`
}

type Contractor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WorkspaceUser struct {
	ID           string  `json:"id"`
	Email        string  `json:"email"`
	DisplayName  string  `json:"displayName"`
	Role         string  `json:"role"`
	Status       string  `json:"status"`
	CustomRoleID *string `json:"customRoleId,omitempty"`
}

type Workspace struct {
	Zones          []Zone          `json:"zones"`
	Contracts      []Contract      `json:"contracts"`
	Engineers      []Engineer      `json:"engineers"`
	WorkspaceUsers []WorkspaceUser `json:"workspaceUsers"`
	Contractors    []Contractor    `json:"contractors"`
}

type TaskItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Col      string `json:"col"`
	Assignee string `json:"assignee"`
}

type TaskProject struct {
	// DBID is the task_projects.id surrogate. Populated on load so per-row
	// mutations can address the right DB row without re-keying by sort_order.
	// Excluded from JSON so the wire shape the UI consumes stays unchanged.
	DBID     int        `json:"-"`
	Name     string     `json:"name"`
	Sections []string   `json:"sections"`
	Tasks    []TaskItem `json:"tasks"`
}

type TasksStore struct {
	Projects []TaskProject `json:"projects"`
}

type Milestone struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Due    string `json:"due"`
	Zone   string `json:"zone"`
	Status string `json:"status"`
	Owner  string `json:"owner"`
}

type AuditEntry struct {
	ID     string `json:"id"`
	At     string `json:"at"`
	User   string `json:"user"`
	Action string `json:"action"`
	Detail string `json:"detail"`
}

type MaterialEntry struct {
	ID       string `json:"id"`
	Item     string `json:"item"`
	Zone     string `json:"zone"`
	Qty      int    `json:"qty"`
	Unit     string `json:"unit"`
	Date     string `json:"date"`
	Supplier string `json:"supplier"`
}

type CustomRole struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Template    *string  `json:"template,omitempty"`
}

type FrontendStore struct {
	Tasks         TasksStore            `json:"tasks"`
	Milestones    []Milestone           `json:"milestones"`
	Assistance    []AssistanceMessage   `json:"assistance"`
	Audit         []AuditEntry          `json:"audit"`
	Materials     []MaterialEntry       `json:"materials"`
	Updates       []any                 `json:"updates"`
	CustomRoles   []CustomRole          `json:"customRoles"`
	ProfilePhotos map[string]string     `json:"profilePhotos"`
	AiScan        any                   `json:"aiScan,omitempty"`
}

type ContractInput struct {
	No      string         `json:"no"`
	Name    string         `json:"name"`
	Zone    string         `json:"zone"`
	Status  ContractStatus `json:"status"`
	Pri     Priority       `json:"pri"`
	Prog    int            `json:"prog"`
	Sup     string         `json:"sup"`
	Remarks string         `json:"remarks"`
	Cs      *int64         `json:"cs,omitempty"`
	Paid    *int64         `json:"paid,omitempty"`
	Workers *int           `json:"workers,omitempty"`
}

type ContractPatch struct {
	Name    *string         `json:"name,omitempty"`
	Zone    *string         `json:"zone,omitempty"`
	Status  *ContractStatus `json:"status,omitempty"`
	Pri     *Priority       `json:"pri,omitempty"`
	Prog    *int            `json:"prog,omitempty"`
	Sup     *string         `json:"sup,omitempty"`
	Remarks *string         `json:"remarks,omitempty"`
	Cs      *int64          `json:"cs,omitempty"`
	Paid    *int64          `json:"paid,omitempty"`
	Workers *int            `json:"workers,omitempty"`
}

// Session is derived from platform JWT claims; the issuer is
// iag-authentication (NOT this service). Permissions come straight from the
// token — this service no longer hosts a user/role/permission tier.
type Session struct {
	Email         string   `json:"email"`
	Role          string   `json:"role"`
	DisplayName   string   `json:"displayName"`
	ContractorSup *string  `json:"contractorSup,omitempty"`
	Permissions   []string `json:"-"`
}

type BootstrapResponse struct {
	Session     Session           `json:"session"`
	Workspace   Workspace         `json:"workspace"`
	Frontend    FrontendStore     `json:"frontend"`
	Permissions PermissionContext `json:"permissions"`
}

// SessionResponse is returned by GET /v1/session.
type SessionResponse struct {
	Session     Session           `json:"session"`
	Permissions PermissionContext `json:"permissions"`
}

type AuditInput struct {
	Action string `json:"action"`
	Detail string `json:"detail"`
}

type AssistanceMessage struct {
	From string `json:"from"`
	Text string `json:"text"`
	At   string `json:"at"`
}

type AssistanceInput struct {
	Text string `json:"text"`
}

type ProfilePhotoInput struct {
	Email   string `json:"email"`
	DataURL string `json:"dataUrl"`
}

type EngineerInput struct {
	Name   string `json:"name"`
	Role   string `json:"role"`
	Zone   string `json:"zone"`
	Phone  string `json:"phone"`
	Email  string `json:"email"`
	Active string `json:"active"`
}

type UserInput struct {
	Email        string  `json:"email"`
	DisplayName  string  `json:"displayName"`
	Role         string  `json:"role"`
	Status       string  `json:"status"`
	CustomRoleID *string `json:"customRoleId"`
}

type UserPatch struct {
	Email        *string `json:"email,omitempty"`
	DisplayName  *string `json:"displayName,omitempty"`
	Role         *string `json:"role,omitempty"`
	Status       *string `json:"status,omitempty"`
	CustomRoleID *string `json:"customRoleId,omitempty"`
}

// ProfilePatch is the self-service body for PATCH /v1/profile (display name only).
type ProfilePatch struct {
	DisplayName *string `json:"displayName,omitempty"`
}

type MilestoneInput struct {
	Title  string `json:"title"`
	Zone   string `json:"zone"`
	Due    string `json:"due"`
	Owner  string `json:"owner"`
	Status string `json:"status"`
}

type MilestonePatch struct {
	Title  *string `json:"title,omitempty"`
	Zone   *string `json:"zone,omitempty"`
	Due    *string `json:"due,omitempty"`
	Owner  *string `json:"owner,omitempty"`
	Status *string `json:"status,omitempty"`
}

type MaterialInput struct {
	Item     string `json:"item"`
	Zone     string `json:"zone"`
	Qty      int    `json:"qty"`
	Unit     string `json:"unit"`
	Supplier string `json:"supplier"`
	Date     string `json:"date"`
}

type MaterialPatch struct {
	Item     *string `json:"item,omitempty"`
	Zone     *string `json:"zone,omitempty"`
	Qty      *int    `json:"qty,omitempty"`
	Unit     *string `json:"unit,omitempty"`
	Supplier *string `json:"supplier,omitempty"`
	Date     *string `json:"date,omitempty"`
}

type ProjectInput struct {
	Name string `json:"name"`
}

type ProjectPatch struct {
	Name     *string   `json:"name,omitempty"`
	Sections *[]string `json:"sections,omitempty"`
}

type ZonePatch struct {
	Name  *string `json:"name,omitempty"`
	Desc  *string `json:"desc,omitempty"`
	Sup   *string `json:"sup,omitempty"`
	Color *string `json:"color,omitempty"`
}

type TaskInput struct {
	Title    string `json:"title"`
	Col      string `json:"col"`
	Assignee string `json:"assignee"`
}

type TaskPatch struct {
	Title    *string `json:"title,omitempty"`
	Col      *string `json:"col,omitempty"`
	Assignee *string `json:"assignee,omitempty"`
}

type RoleInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Template    *string  `json:"template"`
}

type RolePatch struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Template    *string  `json:"template,omitempty"`
}

// UI-facing types mirroring the server domain model.
//
// Field names match the server's JSON exactly (snake_case: created_at,
// project_id, …). The API layer returns these shapes verbatim inside the
// `{ status, message, data }` envelope, so components read snake_case fields
// directly.
//
// The main entities come first; the aggregate/composite types they are made of
// (list pages, dashboard) live at the bottom.

// --- Projects & environments ----------------------------------------------

// Top-level container for secrets.
export interface Project {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
  // Counts are present on list endpoints (GET /projects) but absent on create/detail.
  environment_count?: number;
  tag_count?: number;
}

// A deployment target (e.g. production) inside a project.
export interface Environment {
  id: string;
  project_id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
  // Present on list endpoints (GET /projects/:id/environments), absent on create/detail.
  variable_count?: number;
}

// A label attached to variables for filtering.
export interface Tag {
  id: string;
  project_id: string;
  name: string;
}

// A key inside an environment; its plaintext value lives separately and is
// revealed on demand via GET /variables/:id/value. The server returns the full
// tag objects under `tags`.
export interface Variable {
  id: string;
  environment_id: string;
  name: string;
  created_at: string;
  updated_at: string;
  tags: Tag[];
}

// A point-in-time value; only one version is active at a time.
// The server never returns the plaintext value here (value is always empty),
// so historical values cannot be revealed through the UI.
export interface VariableVersion {
  id: string;
  variable_id: string;
  version: number;
  value: string;
  active: boolean;
  created_at: string;
  updated_at: string;
}

// --- Authentication & tokens ------------------------------------------------

// A client access token. The secret is only ever returned once, at creation.
export interface Token {
  id: string;
  name: string;
  created_at: string;
  expires_at: string;
  revoked_at: string | null;
}

// The one-time creation response that includes the raw "yily_…" token value.
// Its shape differs from `Token` (has `user_id` and `token`, no `revoked_at`).
export interface CreatedToken {
  id: string;
  user_id: string;
  name: string;
  token: string;
  created_at: string;
  expires_at: string;
}

// The signed-in Super Admin.
export interface User {
  id: string;
  name: string;
  email: string;
  created_at?: string;
  updated_at?: string;
}

// --- Audit log ---------------------------------------------------------------

export type ActorType = "user" | "token" | "system";

// A point-in-time audit record: who did what to which resource.
export interface AuditEvent {
  id: string;
  created_at: string;
  actor_type: ActorType;
  actor_id: string;
  action: string;
  resource_type: string;
  resource_id: string;
  resource_name: string;
  metadata: string;
}

// --- Aggregates (list pages & dashboard) ------------------------------------

// A page of audit events with a total count.
export interface AuditList {
  events: AuditEvent[];
  total: number;
}

// Aggregate counts shown at the top of the dashboard.
export interface DashboardStats {
  projects: number;
  environments: number;
  variables: number;
  tokens: number;
}

// A project summary on the dashboard, including environment and variable counts.
export interface ProjectSummary {
  id: string;
  name: string;
  environment_count: number;
  variable_count: number;
  updated_at: string;
}

// The full dashboard payload.
export interface Dashboard {
  stats: DashboardStats;
  recent_activity: AuditEvent[];
  projects: ProjectSummary[];
}

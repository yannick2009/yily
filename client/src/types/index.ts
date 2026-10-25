// UI-facing types mirroring the server domain model.
//
// The API returns snake_case fields (created_at, project_id, …) inside a
// `{ status, message, data }` envelope. Mapping to these camelCase types will
// live in the API layer when we wire the client to the server.

// Top-level container for secrets.
export interface Project {
  id: string;
  name: string;
  createdAt: string;
  updatedAt: string;
}

// A deployment target (e.g. production) inside a project.
export interface Environment {
  id: string;
  projectId: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

// A label attached to variables for filtering.
export interface Tag {
  id: string;
  projectId: string;
  name: string;
}

// A key inside an environment; its plaintext value lives separately.
export interface Variable {
  id: string;
  environmentId: string;
  name: string;
  tagIds: string[];
  createdAt: string;
  updatedAt: string;
}

// A point-in-time value; only one version is active at a time.
export interface VariableVersion {
  id: string;
  variableId: string;
  version: number;
  value: string;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

// A client access token (secret shown once at creation).
export interface Token {
  id: string;
  name: string;
  createdAt: string;
  lastUsedAt: string | null;
}

// The signed-in user (mock until auth is wired to the server).
export interface User {
  id: string;
  name: string;
  email: string;
}

// A point-in-time audit record: who did what to which resource.
export interface AuditEvent {
  id: string;
  createdAt: string;
  actorType: "user" | "token" | "system";
  actorId: string;
  action: string;
  resourceType: string;
  resourceId: string;
  resourceName: string;
  metadata: string;
}

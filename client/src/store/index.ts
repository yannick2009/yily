import { createSignal } from "solid-js";
import { createStore } from "solid-js/store";
import { tokenSecrets, tokens, values, variableVersions } from "./mock";
import { auditEvents, environments, projects, tags, variables } from "./mock";
import type { AuditEvent, Environment, Project, Tag, Token, User, Variable, VariableVersion } from "../types";

interface Db {
  projects: Project[];
  environments: Environment[];
  tags: Tag[];
  variables: Variable[];
  variableVersions: VariableVersion[];
  tokens: Token[];
  auditEvents: AuditEvent[];
}

// App-wide reactive store seeded from the mock data. Swapping the seed data for
// API calls is the only change needed when we wire the client to the server.
export const [db, setDb] = createStore<Db>({
  projects,
  environments,
  tags,
  variables,
  variableVersions,
  tokens,
  auditEvents,
});

// Signed-in user. Empty until the first-run /init screen creates the account.
export const [currentUser, setCurrentUser] = createSignal<User>({
  id: "",
  name: "",
  email: "",
});

// Whether the instance has an account yet. Starts false so a fresh instance
// lands on the /init setup screen instead of the login page.
export const [initialized, setInitialized] = createSignal(false);

// Update the mock profile name/email.
export function updateCurrentUser(patch: Partial<Pick<User, "name" | "email">>): void {
  setCurrentUser((u) => ({ ...u, ...patch }));
}

// Mock account password (plaintext only because this is a mock; the real API
// never exposes it). Used by the settings screen to simulate a password change.
export const [currentPassword, setCurrentPassword] = createSignal("demo1234");

// Change the password after validating the current one. Returns true on success.
export function updatePassword(current: string, next: string): boolean {
  if (current !== currentPassword()) return false;
  setCurrentPassword(next);
  return true;
}

// Create the first admin account during first-run setup.
export function initAccount(name: string, email: string, password: string): void {
  setCurrentUser({ id: "u1", name: name.trim(), email: email.trim() });
  setCurrentPassword(password);
  setInitialized(true);
}

// Generate a UUID (fallback to a random string) and an ISO timestamp.
const uid = () =>
  typeof crypto !== "undefined" && crypto.randomUUID
    ? crypto.randomUUID()
    : Math.random().toString(36).slice(2);
const now = () => new Date().toISOString();

// Create a project and return its new id.
export function addProject(name: string): string {
  const id = uid();
  setDb("projects", (p) => [...p, { id, name, createdAt: now(), updatedAt: now() }]);
  return id;
}

// Update a project's name.
export function renameProject(id: string, name: string): void {
  setDb("projects", (p) => p.map((x) => (x.id === id ? { ...x, name, updatedAt: now() } : x)));
}

// Delete a project and cascade to its environments, variables and versions.
export function removeProject(id: string): void {
  const envIds = db.environments.filter((e) => e.projectId === id).map((e) => e.id);
  const varIds = db.variables.filter((v) => envIds.includes(v.environmentId)).map((v) => v.id);
  setDb("environments", (e) => e.filter((x) => x.projectId !== id));
  setDb("tags", (t) => t.filter((x) => x.projectId !== id));
  setDb("variables", (v) => v.filter((x) => !envIds.includes(x.environmentId)));
  setDb("variableVersions", (v) => v.filter((x) => !varIds.includes(x.variableId)));
  varIds.forEach((vid) => delete values[vid]);
  setDb("projects", (p) => p.filter((x) => x.id !== id));
}

// Wipe every project and all related data (danger-zone rescue).
export function clearAllProjects(): void {
  setDb("environments", []);
  setDb("tags", []);
  setDb("variables", []);
  setDb("variableVersions", []);
  setDb("projects", []);
  Object.keys(values).forEach((key) => delete values[key]);
}

// Create an environment inside a project and return its id.
export function addEnvironment(projectId: string, name: string, description: string): string {
  const id = uid();
  setDb("environments", (e) => [
    ...e,
    { id, projectId, name, description, createdAt: now(), updatedAt: now() },
  ]);
  return id;
}

// Delete an environment and cascade to its variables and versions.
export function removeEnvironment(id: string): void {
  const varIds = db.variables.filter((v) => v.environmentId === id).map((v) => v.id);
  setDb("variables", (v) => v.filter((x) => x.environmentId !== id));
  setDb("variableVersions", (v) => v.filter((x) => !varIds.includes(x.variableId)));
  varIds.forEach((vid) => delete values[vid]);
  setDb("environments", (e) => e.filter((x) => x.id !== id));
}

// Update an environment's name.
export function renameEnvironment(id: string, name: string): void {
  setDb("environments", (e) =>
    e.map((x) => (x.id === id ? { ...x, name: name.trim(), updatedAt: now() } : x)),
  );
}

// Update an environment's description.
export function setEnvironmentDescription(id: string, description: string): void {
  setDb("environments", (e) =>
    e.map((x) => (x.id === id ? { ...x, description: description.trim(), updatedAt: now() } : x)),
  );
}

// Create a tag inside a project and return its id.
export function addTag(projectId: string, name: string): string {
  const id = uid();
  setDb("tags", (t) => [...t, { id, projectId, name }]);
  return id;
}

// Delete a tag and detach it from every variable in the project.
export function removeTag(projectId: string, tagId: string): void {
  void projectId;
  setDb("tags", (t) => t.filter((x) => x.id !== tagId));
  setDb("variables", (v) =>
    v.map((x) =>
      x.tagIds.includes(tagId) ? { ...x, tagIds: x.tagIds.filter((t) => t !== tagId) } : x,
    ),
  );
}

// Create a variable (name uppercased); a value also creates its first version.
export function addVariable(environmentId: string, name: string, value: string): string {
  const id = uid();
  const ts = now();
  setDb("variables", (v) => [
    ...v,
    {
      id,
      environmentId,
      name: name.trim().toUpperCase(),
      tagIds: [],
      createdAt: ts,
      updatedAt: ts,
    },
  ]);
  if (value.trim() !== "") {
    values[id] = value.trim();
    setDb("variableVersions", (v) => [
      ...v,
      {
        id: uid(),
        variableId: id,
        version: 1,
        value: value.trim(),
        active: true,
        createdAt: ts,
        updatedAt: ts,
      },
    ]);
  }
  return id;
}

// Delete a variable and its version history.
export function removeVariable(id: string): void {
  setDb("variables", (v) => v.filter((x) => x.id !== id));
  setDb("variableVersions", (v) => v.filter((x) => x.variableId !== id));
  delete values[id];
}

// Update a variable's name (uppercased).
export function renameVariable(id: string, name: string): void {
  setDb("variables", (v) =>
    v.map((x) => (x.id === id ? { ...x, name: name.trim().toUpperCase(), updatedAt: now() } : x)),
  );
}

// Replace a variable's tag ids.
export function setVariableTags(variableId: string, tagIds: string[]): void {
  setDb("variables", (v) =>
    v.map((x) => (x.id === variableId ? { ...x, tagIds, updatedAt: now() } : x)),
  );
}

// Deactivate every version and remove the current plaintext value.
export function clearVariableValue(variableId: string): void {
  setDb("variableVersions", (v) =>
    v.map((x) => (x.variableId === variableId ? { ...x, active: false, updatedAt: now() } : x)),
  );
  delete values[variableId];
  setDb("variables", (v) => v.map((x) => (x.id === variableId ? { ...x, updatedAt: now() } : x)));
}

// Append a new (active) version and update the current plaintext value.
export function addVariableVersion(variableId: string, value: string): void {
  const next =
    db.variableVersions
      .filter((v) => v.variableId === variableId)
      .reduce((m, v) => Math.max(m, v.version), 0) + 1;
  const ts = now();
  setDb("variableVersions", (v) => [
    ...v,
    {
      id: uid(),
      variableId,
      version: next,
      value: value.trim(),
      active: true,
      createdAt: ts,
      updatedAt: ts,
    },
  ]);
  values[variableId] = value.trim();
  setDb("variables", (v) => v.map((x) => (x.id === variableId ? { ...x, updatedAt: ts } : x)));
}

// Activate/deactivate a version and recompute the current plaintext value.
export function setVariableVersionActive(versionId: string, active: boolean): void {
  setDb("variableVersions", (v) =>
    v.map((x) => (x.id === versionId ? { ...x, active, updatedAt: now() } : x)),
  );
  const vv = db.variableVersions.find((x) => x.id === versionId);
  if (vv) {
    const highest = db.variableVersions
      .filter((x) => x.variableId === vv.variableId && x.active)
      .sort((a, b) => b.version - a.version)[0];
    if (highest) values[vv.variableId] = highest.value;
    else delete values[vv.variableId];
  }
}

// Bulk import name→value entries: create new variables, version the changed ones.
export function importVariables(environmentId: string, entries: Record<string, string>): void {
  for (const [name, value] of Object.entries(entries)) {
    const normalized = name.trim().toUpperCase();
    const trimmed = value.trim();
    const existing = db.variables.find(
      (x) => x.environmentId === environmentId && x.name === normalized,
    );
    if (existing) {
      if (trimmed !== (values[existing.id] ?? "")) addVariableVersion(existing.id, trimmed);
    } else {
      addVariable(environmentId, normalized, trimmed);
    }
  }
}

// Generate a random token secret (prefixed yily_live_).
function randomSecret(): string {
  const chars = "0123456789abcdef";
  let out = "";
  for (let i = 0; i < 32; i++) out += chars[Math.floor(Math.random() * chars.length)];
  return `yily_live_${out}`;
}

// Create a token, store its secret once, and return its id.
export function addToken(name: string): string {
  const id = uid();
  tokenSecrets[id] = randomSecret();
  setDb("tokens", (t) => [...t, { id, name, createdAt: now(), lastUsedAt: null }]);
  return id;
}

// Delete a token and its secret.
export function removeToken(id: string): void {
  setDb("tokens", (t) => t.filter((x) => x.id !== id));
  delete tokenSecrets[id];
}

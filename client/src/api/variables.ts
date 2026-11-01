// Variables and their versions API.
import { request } from "./client";
import type { Variable, VariableVersion } from "../types";

// Options for listing variables: both filters are applied server-side.
export interface ListVariablesOptions {
  tagIds?: string[];
  search?: string;
}

// list returns the variables of an environment, optionally filtered by tags
// and/or a case-insensitive substring match on the name (both server-side).
export async function list(
  environmentId: string,
  options: ListVariablesOptions = {},
): Promise<Variable[]> {
  const params = new URLSearchParams();
  if (options.tagIds?.length) params.set("tag_ids", options.tagIds.join(","));
  if (options.search?.trim()) params.set("search", options.search.trim());
  const query = params.toString();
  return (
    (await request<Variable[]>(
      "GET",
      `/environments/${environmentId}/variables${query ? `?${query}` : ""}`,
    )) ?? []
  );
}

// create creates a variable (and its first version when a value is provided).
export async function create(
  environmentId: string,
  input: { name: string; value: string },
): Promise<Variable> {
  return request<Variable>("POST", `/environments/${environmentId}/variables`, {
    body: JSON.stringify(input),
  });
}

// get returns a single variable (with its tags).
export async function get(id: string): Promise<Variable> {
  return request<Variable>("GET", `/variables/${id}`);
}

// getValue reveals the variable's current decrypted value.
export async function getValue(id: string): Promise<string> {
  const raw = await request<{ value: string }>("GET", `/variables/${id}/value`);
  return raw.value;
}

// rename updates a variable's name.
export async function rename(id: string, name: string): Promise<void> {
  await request<unknown>("PATCH", `/variables/${id}`, { body: JSON.stringify({ name }) });
}

// remove deletes a variable and its version history.
export async function remove(id: string): Promise<void> {
  await request<unknown>("DELETE", `/variables/${id}`);
}

// setTags replaces the set of tags on a variable.
export async function setTags(id: string, tagIds: string[]): Promise<void> {
  await request<unknown>("PUT", `/variables/${id}/tags`, {
    body: JSON.stringify({ tag_ids: tagIds }),
  });
}

// addVersion appends a new active version to a variable.
export async function addVersion(id: string, value: string): Promise<VariableVersion> {
  return request<VariableVersion>("POST", `/variables/${id}/versions`, {
    body: JSON.stringify({ value }),
  });
}

// listVersions returns all versions of a variable, newest first (values are never exposed).
export async function listVersions(id: string): Promise<VariableVersion[]> {
  return (await request<VariableVersion[]>("GET", `/variables/${id}/versions`)) ?? [];
}

// getCurrentVersion returns the variable's active version (value redacted).
export async function getCurrentVersion(id: string): Promise<VariableVersion> {
  return request<VariableVersion>("GET", `/variables/${id}/versions/current`);
}

// setVersionActive activates or deactivates a version.
export async function setVersionActive(
  variableId: string,
  versionId: string,
  active: boolean,
): Promise<void> {
  await request<unknown>("PATCH", `/variables/${variableId}/versions/${versionId}`, {
    body: JSON.stringify({ active }),
  });
}

// clearValue deactivates every version of a variable, removing its current value.
export async function clearValue(id: string): Promise<void> {
  await request<unknown>("DELETE", `/variables/${id}/value`);
}

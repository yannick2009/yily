// Environments API (including import/export).
import { request, requestRaw } from "./client";
import type { Environment } from "../types";

// list returns every environment of a project with its variable counts.
export async function list(projectId: string): Promise<Environment[]> {
  return (await request<Environment[]>("GET", `/projects/${projectId}/environments`)) ?? [];
}

// create creates an environment inside a project.
export async function create(
  projectId: string,
  input: { name: string; description: string },
): Promise<Environment> {
  return request<Environment>("POST", `/projects/${projectId}/environments`, {
    body: JSON.stringify(input),
  });
}

// get returns a single environment by id.
export async function get(id: string): Promise<Environment> {
  return request<Environment>("GET", `/environments/${id}`);
}

// rename updates an environment's name.
export async function rename(id: string, name: string): Promise<void> {
  await request<unknown>("PATCH", `/environments/${id}/name`, { body: JSON.stringify({ name }) });
}

// setDescription updates an environment's description.
export async function setDescription(id: string, description: string): Promise<void> {
  await request<unknown>("PATCH", `/environments/${id}/description`, {
    body: JSON.stringify({ description }),
  });
}

// remove deletes an environment and its variables.
export async function remove(id: string): Promise<void> {
  await request<unknown>("DELETE", `/environments/${id}`);
}

// importFile uploads a .env/.json/.yaml file to import its variables.
export async function importFile(id: string, file: File): Promise<void> {
  const form = new FormData();
  form.append("file", file);
  await request<unknown>("POST", `/environments/${id}/imports`, { body: form });
}

// exportFile downloads the environment's variables in the requested format and
// returns the raw bytes plus a suggested filename.
export async function exportFile(
  id: string,
  format: "json" | "yaml" | "env",
): Promise<{ blob: Blob; filename: string }> {
  const res = await requestRaw(`/environments/${id}/export?format=${format}`);
  const blob = await res.blob();

  // The server sets Content-Disposition with a sanitized filename; fall back to
  // a generic name if it is missing.
  const disposition = res.headers.get("Content-Disposition") ?? "";
  const match = /filename="([^"]+)"/.exec(disposition);
  const filename = match?.[1] ?? `export.${format === "yaml" ? "yaml" : format}`;

  return { blob, filename };
}

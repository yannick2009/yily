// Tags API.
import { request } from "./client";
import type { CreateTagsResult, Tag } from "../types";

// list returns every tag of a project.
export async function list(projectId: string): Promise<Tag[]> {
  return (await request<Tag[]>("GET", `/projects/${projectId}/tags`)) ?? [];
}

// create creates one or several tags for a project in a single request.
// Server trims, deduplicates, and silently skips names that already exist.
export async function create(projectId: string, names: string[]): Promise<CreateTagsResult> {
  const result = await request<CreateTagsResult>("POST", `/projects/${projectId}/tags`, {
    body: JSON.stringify({ names }),
  });
  return { created: result?.created ?? [], skipped: result?.skipped ?? [] };
}

// remove deletes a tag of a project.
export async function remove(projectId: string, tagId: string): Promise<void> {
  await request<unknown>("DELETE", `/projects/${projectId}/tags/${tagId}`);
}

// Tags API.
import { request } from "./client";
import type { Tag } from "../types";

// list returns every tag of a project.
export async function list(projectId: string): Promise<Tag[]> {
  return (await request<Tag[]>("GET", `/projects/${projectId}/tags`)) ?? [];
}

// create creates a tag for a project.
export async function create(projectId: string, name: string): Promise<Tag> {
  return request<Tag>("POST", `/projects/${projectId}/tags`, {
    body: JSON.stringify({ name }),
  });
}

// remove deletes a tag of a project.
export async function remove(projectId: string, tagId: string): Promise<void> {
  await request<unknown>("DELETE", `/projects/${projectId}/tags/${tagId}`);
}

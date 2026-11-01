// Projects API.
import { request } from "./client";
import type { Project } from "../types";

// list returns every project with its environment and tag counts.
export async function list(): Promise<Project[]> {
  return (await request<Project[]>("GET", "/projects")) ?? [];
}

// create creates a project (and its default environment).
export async function create(input: { name: string }): Promise<Project> {
  return request<Project>("POST", "/projects", { body: JSON.stringify(input) });
}

// get returns a single project by id.
export async function get(id: string): Promise<Project> {
  return request<Project>("GET", `/projects/${id}`);
}

// rename updates a project's name.
export async function rename(id: string, name: string): Promise<void> {
  await request<unknown>("PATCH", `/projects/${id}`, { body: JSON.stringify({ name }) });
}

// remove deletes a project and all of its data.
export async function remove(id: string): Promise<void> {
  await request<unknown>("DELETE", `/projects/${id}`);
}

// clear deletes every project.
export async function clear(): Promise<void> {
  await request<unknown>("DELETE", "/projects");
}

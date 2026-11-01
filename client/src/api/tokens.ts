// Personal access tokens API.
import { request } from "./client";
import type { CreatedToken, Token } from "../types";

// list returns every token owned by the authenticated user (no secrets).
export async function list(): Promise<Token[]> {
  return (await request<Token[]>("GET", "/tokens")) ?? [];
}

// create issues a new token and returns its one-time secret.
export async function create(input: { name: string; expiresInDays?: number }): Promise<CreatedToken> {
  return request<CreatedToken>("POST", "/tokens", { body: JSON.stringify(input) });
}

// revoke invalidates a token immediately.
export async function revoke(id: string): Promise<void> {
  await request<unknown>("DELETE", `/tokens/${id}`);
}

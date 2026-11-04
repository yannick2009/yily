// Personal access tokens API.
import { request } from "./client";
import type { CreatedToken, Token, TokenScope } from "../types";

// list returns every token owned by the authenticated user (no secrets).
export async function list(): Promise<Token[]> {
  return (await request<Token[]>("GET", "/tokens")) ?? [];
}

// create issues a new token and returns its one-time secret. Pass expires_in_days: null for a non-expiring token (read scope only).
export async function create(input: {
  name: string;
  scope: TokenScope;
  expires_in_days?: number | null;
}): Promise<CreatedToken> {
  return request<CreatedToken>("POST", "/tokens", {
    body: JSON.stringify(input),
  });
}

// rotate revokes an existing token and issues a replacement under the same name and scope.
// The new raw secret is returned exactly once. Pass expires_in_days: null to issue a non-expiring replacement (read scope only).
export async function rotate(
  id: string,
  input: { expires_in_days?: number | null },
): Promise<CreatedToken> {
  return request<CreatedToken>("POST", `/tokens/${id}/rotate`, {
    body: JSON.stringify(input),
  });
}

// revoke invalidates a token immediately.
export async function revoke(id: string): Promise<void> {
  await request<unknown>("DELETE", `/tokens/${id}`);
}

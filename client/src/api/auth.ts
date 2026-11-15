// Authentication API: first-run setup, login, logout and the current session.
import { request, setCSRFToken } from "./client";
import type { User } from "../types";

export interface SetupStatus {
  initialized: boolean;
}

// LoginResult is the server's login payload: the user plus the fresh CSRF token.
interface LoginResult {
  user: User;
  csrf_token: string;
}

// status reports whether the instance has a Super Admin account yet.
export async function status(): Promise<SetupStatus> {
  return request<SetupStatus>("GET", "/auth/status");
}

// setup creates the initial Super Admin account, gated by the out-of-band setup token.
export async function setup(input: {
  name: string;
  email: string;
  password: string;
  confirm: string;
  setup_token: string;
}): Promise<User> {
  return request<User>("POST", "/auth/setup", { body: JSON.stringify(input) });
}

// login authenticates, sets the session cookie and stores the returned CSRF
// token for subsequent mutations.
export async function login(input: { email: string; password: string }): Promise<User> {
  const result = await request<LoginResult>("POST", "/auth/login", {
    body: JSON.stringify(input),
  });
  setCSRFToken(result.csrf_token);
  return result.user;
}

// fetchCSRFToken retrieves the current session's CSRF token after a page reload.
export async function fetchCSRFToken(): Promise<void> {
  const result = await request<{ csrf_token: string }>("GET", "/auth/csrf");
  setCSRFToken(result.csrf_token);
}

// logout revokes the current session and clears the cookie.
export async function logout(): Promise<void> {
  await request<unknown>("POST", "/auth/logout");
}

// me returns the authenticated Super Admin from the session.
export async function me(): Promise<User> {
  return request<User>("GET", "/auth/me");
}

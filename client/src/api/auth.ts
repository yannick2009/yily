// Authentication API: first-run setup, login, logout and the current session.
import { request } from "./client";
import type { User } from "../types";

export interface SetupStatus {
  initialized: boolean;
}

// status reports whether the instance has a Super Admin account yet.
export async function status(): Promise<SetupStatus> {
  return request<SetupStatus>("GET", "/auth/status");
}

// setup creates the initial Super Admin account.
export async function setup(input: {
  name: string;
  email: string;
  password: string;
}): Promise<User> {
  return request<User>("POST", "/auth/setup", { body: JSON.stringify(input) });
}

// login authenticates and sets the session cookie.
export async function login(input: { email: string; password: string }): Promise<User> {
  return request<User>("POST", "/auth/login", { body: JSON.stringify(input) });
}

// logout revokes the current session and clears the cookie.
export async function logout(): Promise<void> {
  await request<unknown>("POST", "/auth/logout");
}

// me returns the authenticated Super Admin from the session.
export async function me(): Promise<User> {
  return request<User>("GET", "/auth/me");
}

// Authentication API: first-run setup, login, logout and the current session.
import { request, requestRaw, setCSRFToken } from "./client";
import { downloadBlob } from "../lib/download";
import type {
  ChangePasswordInput,
  LoginResult,
  SetupStatus,
  TOTPSetup,
  UpdateProfileInput,
  User,
} from "../types";

// status reports whether the instance has a Super Admin account yet.
export async function status(): Promise<SetupStatus> {
  return request<SetupStatus>("GET", "/auth/status");
}

// setup creates the initial Super Admin account, gated by the out-of-band setup
// token. On success the server streams the one-time recovery code as a file,
// which we trigger as a browser download.
export async function setup(input: {
  name: string;
  email: string;
  password: string;
  confirm: string;
  setup_token: string;
}): Promise<void> {
  const res = await requestRaw("/auth/setup", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  await downloadRecoveryCode(res);
}

// recover resets the Super Admin password using the recovery code. On success a
// fresh recovery code is downloaded again (the old one is invalidated).
export async function recover(input: {
  recovery_code: string;
  password: string;
  confirm: string;
}): Promise<void> {
  const res = await requestRaw("/auth/recover", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  await downloadRecoveryCode(res);
}

// downloadRecoveryCode saves the recovery-code file returned by the server.
async function downloadRecoveryCode(res: Response): Promise<void> {
  const blob = await res.blob();
  const disposition = res.headers.get("Content-Disposition") ?? "";
  const match = /filename="([^"]+)"/.exec(disposition);
  downloadBlob(blob, match?.[1] ?? "yily-recovery-code.txt");
}

// LoginInput carries the password and, when TOTP is enabled, the code from the
// authenticator app.
export interface LoginInput {
  email: string;
  password: string;
  totp_code?: string;
}

// login authenticates with a password (plus an optional TOTP code), sets the
// session cookie and stores the returned CSRF token for subsequent mutations.
export async function login(input: LoginInput): Promise<User> {
  return completeLogin("/auth/login", input);
}

// loginWithTOTP authenticates with an authenticator code alone.
export async function loginWithTOTP(input: { email: string; code: string }): Promise<User> {
  return completeLogin("/auth/login/totp", input);
}

// completeLogin performs a login request and stores the returned CSRF token.
async function completeLogin(path: string, input: unknown): Promise<User> {
  const result = await request<LoginResult>("POST", path, {
    body: JSON.stringify(input),
  });
  setCSRFToken(result.csrf_token);
  return result.user;
}

// totpSetup starts enrollment and returns the seed and provisioning data.
export async function totpSetup(): Promise<TOTPSetup> {
  return request<TOTPSetup>("POST", "/auth/totp/setup");
}

// totpConfirm verifies a code against the pending seed and enables TOTP.
export async function totpConfirm(code: string): Promise<void> {
  await request<{ enabled: boolean }>("POST", "/auth/totp/confirm", {
    body: JSON.stringify({ code }),
  });
}

// totpDisable turns TOTP off after re-verifying the account password.
export async function totpDisable(password: string): Promise<void> {
  await request<{ enabled: boolean }>("POST", "/auth/totp/disable", {
    body: JSON.stringify({ password }),
  });
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

// updateProfile updates the signed-in Super Admin's display name and/or email
// and returns the refreshed account.
export async function updateProfile(input: UpdateProfileInput): Promise<User> {
  return request<User>("PATCH", "/auth/me", {
    body: JSON.stringify(input),
  });
}

// changePassword replaces the signed-in Super Admin's password. The server
// revokes every other session; the current session stays usable.
export async function changePassword(input: ChangePasswordInput): Promise<void> {
  await request<unknown>("PATCH", "/auth/me/password", {
    body: JSON.stringify(input),
  });
}

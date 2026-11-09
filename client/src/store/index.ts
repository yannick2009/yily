// App-wide auth state backed by the real API.
//
// Data (projects, variables, …) is no longer stored here — each route fetches
// what it needs through the API layer and keeps it in a createResource. This
// store holds boot/auth state and cached server metadata shared across the app.

import { createSignal } from "solid-js";
import { auth, system } from "../api";
import type { UpdateInfo, User, VersionInfo } from "../types";

// The signed-in Super Admin, or null when there is no active session.
export const [currentUser, setCurrentUser] = createSignal<User | null>(null);

// Whether the instance has an account yet. null means "not determined yet".
export const [initialized, setInitialized] = createSignal<boolean | null>(null);

// Version of the server, fetched once after authentication.
export const [serverVersion, setServerVersion] = createSignal<VersionInfo | null>(null);

// Latest release information, loaded once after authentication.
export const [updateInfo, setUpdateInfo] = createSignal<UpdateInfo | null>(null);

async function loadVersion(): Promise<void> {
  try {
    setServerVersion(await system.getVersion());
  } catch {
    setServerVersion(null);
  }
}

async function loadUpdate(): Promise<void> {
  try {
    setUpdateInfo(await system.getUpdate());
  } catch {
    setUpdateInfo(null);
  }
}

// boot determines the auth state once at startup: whether the instance is
// initialized and, if so, whether the current session is valid. Version and
// update information are fetched only once the session is known to be valid.
export async function boot(): Promise<void> {
  try {
    const status = await auth.status();
    setInitialized(status.initialized);

    if (!status.initialized) {
      setCurrentUser(null);
      return;
    }

    try {
      setCurrentUser(await auth.me());
    } catch {
      setCurrentUser(null);
      return;
    }

    void loadUpdate();
    await loadVersion();
  } catch {
    setInitialized(false);
    setCurrentUser(null);
  }
}

// setup creates the initial Super Admin account. It does not sign the new
// admin in — the setup endpoint returns no session cookie, so the user must
// log in afterwards. We only mark the instance as initialized.
export async function setup(name: string, email: string, password: string): Promise<void> {
  await auth.setup({ name, email, password });
  setCurrentUser(null);
  setInitialized(true);
}

// login signs the Super Admin in and stores the returned user.
export async function login(email: string, password: string): Promise<void> {
  const user = await auth.login({ email, password });
  setCurrentUser(user);
  setInitialized(true);
  void loadUpdate();
  await loadVersion();
}

// logout revokes the server session and clears the local user state. The
// local state is cleared even when the server call fails (e.g. the session
// already expired), so the UI always signs out and the router guard redirects.
export async function logout(): Promise<void> {
  try {
    await auth.logout();
  } catch {
    // Session is already gone (or the network failed); clear local state anyway.
  }
  setCurrentUser(null);
  setServerVersion(null);
  setUpdateInfo(null);
}

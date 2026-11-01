// App-wide auth state backed by the real API.
//
// Data (projects, variables, …) is no longer stored here — each route fetches
// what it needs through the API layer and keeps it in a createResource. This
// store only holds the boot/auth state shared across the shell and login/init.

import { createSignal } from "solid-js";
import { auth } from "../api";
import type { User } from "../types";

// The signed-in Super Admin, or null when there is no active session.
export const [currentUser, setCurrentUser] = createSignal<User | null>(null);

// Whether the instance has an account yet. null means "not determined yet".
export const [initialized, setInitialized] = createSignal<boolean | null>(null);

// boot determines the auth state once at startup: whether the instance is
// initialized and, if so, whether the current session is valid.
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
    }
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
}

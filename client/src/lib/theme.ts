// Theme state (system/light/dark): reactive signal + localStorage persistence,
// applied to <html data-theme> before first paint.
import { createSignal } from "solid-js";

export type Theme = "system" | "light" | "dark";

const STORAGE_KEY = "yily-theme";
const ORDER: Theme[] = ["system", "light", "dark"];

// Read the persisted theme from localStorage (defaults to system).
function readStored(): Theme {
  try {
    const v = localStorage.getItem(STORAGE_KEY);
    if (v === "light" || v === "dark" || v === "system") return v;
  } catch {
    /* localStorage unavailable */
  }
  return "system";
}

// Apply immediately (before paint the inline script in index.html does the same)
// so there is no flash of the wrong theme.
const initial = readStored();
document.documentElement.setAttribute("data-theme", initial);

// Reactive theme signal shared across the app.
export const [theme, setTheme] = createSignal<Theme>(initial);

// Persist a theme, set <html data-theme>, and refresh the signal.
export function applyTheme(next: Theme): void {
  document.documentElement.setAttribute("data-theme", next);
  try {
    localStorage.setItem(STORAGE_KEY, next);
  } catch {
    /* localStorage unavailable */
  }
  setTheme(next);
}

// Advance to the next theme (system → light → dark → …).
export function cycleTheme(): void {
  applyTheme(ORDER[(ORDER.indexOf(theme()) + 1) % ORDER.length]);
}

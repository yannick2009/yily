// Main layout: sidebar navigation + content area rendering the route children.
import { createMemo } from "solid-js";
import { A, type RouteSectionProps } from "@solidjs/router";
import { t } from "../lib/i18n";
import { cycleTheme, theme } from "../lib/theme";
import { Icon, type IconName } from "../components/ui/Icon";
import { currentUser, logout } from "../store";

export function AppShell(props: RouteSectionProps) {
  // Avatar initials from the user's name ("Jane Doe" → "JD").
  const initials = createMemo(() =>
    (currentUser()?.name ?? "")
      .split(" ")
      .map((part) => part[0] ?? "")
      .join("")
      .toUpperCase(),
  );
  return (
    <div class="flex h-screen bg-bg text-ink">
      <aside class="flex w-60 shrink-0 flex-col border-r border-line bg-surface">
        <div class="flex h-30 items-center justify-center px-5">
          <img src="/logo.svg" alt="Yily" class="h-14 w-auto" />
        </div>

        <nav class="flex-1 space-y-1 px-2 py-3">
          <p class="px-3 pb-2 text-xs font-medium uppercase tracking-wide text-ink-faint">
            {t("nav.workspace")}
          </p>
          <A
            href="/"
            end
            class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium"
            inactiveClass="text-ink-muted hover:bg-surface-2 hover:text-ink"
            activeClass="bg-brand text-on-brand"
          >
            <Icon name="squares-2x2" size={15} />
            {t("dashboard.title")}
          </A>
          <A
            href="/projects"
            class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium"
            inactiveClass="text-ink-muted hover:bg-surface-2 hover:text-ink"
            activeClass="bg-brand text-on-brand"
          >
            <Icon name="folder" size={15} />
            {t("nav.projects")}
          </A>
          <A
            href="/tokens"
            class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium"
            inactiveClass="text-ink-muted hover:bg-surface-2 hover:text-ink"
            activeClass="bg-brand text-on-brand"
          >
            <Icon name="key" size={15} />
            {t("nav.tokens")}
          </A>
          <A
            href="/audit"
            class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium"
            inactiveClass="text-ink-muted hover:bg-surface-2 hover:text-ink"
            activeClass="bg-brand text-on-brand"
          >
            <Icon name="clock" size={15} />
            {t("audit.title")}
          </A>
        </nav>

        <div class="border-t border-line p-2">
          <div class="space-y-1">
            <A
              href="/settings"
              class="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium"
              inactiveClass="text-ink-muted hover:bg-surface-2 hover:text-ink"
              activeClass="bg-brand text-on-brand"
            >
              <Icon name="settings" size={15} />
              {t("nav.settings")}
            </A>
            <button
              type="button"
              onClick={cycleTheme}
              class="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium text-ink-muted hover:bg-surface-2 hover:text-ink"
              title={`${t("nav.theme")}: ${t(`theme.${theme()}`)}`}
            >
              <Icon name={themeIcon()} size={15} />
              <span class="flex-1 text-left">{t("nav.theme")}</span>
              <span class="text-xs text-ink-faint">{t(`theme.${theme()}`)}</span>
            </button>
          </div>
        </div>

        <div class="border-t border-line p-2">
          <div class="flex items-center gap-1">
            <A
              href="/profile"
              class="flex min-w-0 flex-1 items-center gap-2.5 rounded-lg px-2 py-1.5 hover:bg-surface-2"
            >
              <span
                class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-brand text-xs font-semibold text-on-brand"
                aria-hidden="true"
              >
                {initials()}
              </span>
              <div class="min-w-0 flex-1">
                <p class="truncate text-sm font-medium text-ink">{currentUser()?.name}</p>
                <p class="truncate text-xs text-ink-muted">{currentUser()?.email}</p>
              </div>
            </A>
            <button
              type="button"
              onClick={() => void logout()}
              class="rounded-md p-1.5 text-ink-muted hover:bg-surface-2 hover:text-ink"
              title={t("nav.signOut")}
              aria-label={t("nav.signOut")}
            >
              <Icon name="logout" size={15} />
            </button>
          </div>
        </div>
      </aside>

      <main class="flex-1 overflow-y-auto">
        <div class="mx-auto w-full max-w-5xl px-6 py-8">{props.children}</div>
      </main>
    </div>
  );
}

// Theme icon: sun for light, moon for dark, desktop for system.
const themeIcon = (): IconName =>
  theme() === "light" ? "sun" : theme() === "dark" ? "moon" : "computer-desktop";

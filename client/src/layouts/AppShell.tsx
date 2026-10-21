// Main layout: sidebar navigation + content area rendering the route children.
import { A, type RouteSectionProps } from "@solidjs/router";
import { t } from "../lib/i18n";
import { cycleTheme, theme } from "../lib/theme";
import { Icon, type IconName } from "../components/ui/Icon";

// AppShell component provides the main layout for the application, including a sidebar with navigation links and a content area for rendering route-specific components. It handles theme switching and displays the current theme in the sidebar.
// The sidebar includes links to the projects, tokens, and settings pages, and it highlights the active route. The content area is scrollable and centered within the layout.
export function AppShell(props: RouteSectionProps) {
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
            href="/projects"
            class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium"
            inactiveClass="text-ink-muted hover:bg-surface-2 hover:text-ink"
            activeClass="bg-brand text-on-brand">
            <Icon name="folder" size={15} />
            {t("nav.projects")}
          </A>
          <A
            href="/tokens"
            class="flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium"
            inactiveClass="text-ink-muted hover:bg-surface-2 hover:text-ink"
            activeClass="bg-brand text-on-brand">
            <Icon name="key" size={15} />
            {t("nav.tokens")}
          </A>
        </nav>

        <div class="border-t border-line p-2">
          <div class="space-y-1">
            <A
              href="/settings"
              class="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium"
              inactiveClass="text-ink-muted hover:bg-surface-2 hover:text-ink"
              activeClass="bg-brand text-on-brand">
              <Icon name="settings" size={15} />
              {t("nav.settings")}
            </A>
            <button
              type="button"
              onClick={cycleTheme}
              class="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium text-ink-muted hover:bg-surface-2 hover:text-ink"
              title={`${t("nav.theme")}: ${t(`theme.${theme()}`)}`}>
              <Icon name={themeIcon()} size={15} />
              <span class="flex-1 text-left">{t("nav.theme")}</span>
              <span class="text-xs text-ink-faint">
                {t(`theme.${theme()}`)}
              </span>
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

// Function to determine the appropriate icon for the current theme. It returns a sun icon for light theme, a moon icon for dark theme, and a computer desktop icon for system theme.
const themeIcon = (): IconName =>
  theme() === "light"
    ? "sun"
    : theme() === "dark"
      ? "moon"
      : "computer-desktop";

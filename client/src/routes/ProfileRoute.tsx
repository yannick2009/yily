// Dedicated profile page: account identity and sign-out.
import { useNavigate } from "@solidjs/router";
import { Button } from "../components/ui";
import { t } from "../lib/i18n";
import { currentUser, logout } from "../store";

export function ProfileRoute() {
  const navigate = useNavigate();

  const initials = () =>
    (currentUser()?.name ?? "")
      .split(" ")
      .map((part) => part[0] ?? "")
      .join("")
      .toUpperCase();

  const signOut = async () => {
    try {
      await logout();
    } catch {
      /* ignore; the boot gate will still clear the session on the next check */
    }
    navigate("/login", { replace: true });
  };

  return (
    <div class="mx-auto max-w-xl space-y-6">
      <div class="flex items-center gap-4">
        <span
          class="flex h-16 w-16 shrink-0 items-center justify-center rounded-full bg-brand text-xl font-semibold text-on-brand"
          aria-hidden="true"
        >
          {initials()}
        </span>
        <div class="min-w-0 flex-1">
          <h1 class="truncate text-xl font-semibold text-ink">{currentUser()?.name ?? ""}</h1>
          <p class="truncate text-sm text-ink-muted">{currentUser()?.email ?? ""}</p>
        </div>
      </div>

      <section class="rounded-xl border border-line bg-surface p-4">
        <h2 class="text-sm font-semibold text-ink">{t("settings.profile")}</h2>
        <dl class="mt-4 space-y-3 text-sm">
          <div class="flex items-start justify-between gap-4">
            <dt class="shrink-0 text-xs font-medium uppercase tracking-wide text-ink-faint">
              {t("common.name")}
            </dt>
            <dd class="min-w-0 text-right text-ink">{currentUser()?.name ?? ""}</dd>
          </div>
          <div class="flex items-start justify-between gap-4">
            <dt class="shrink-0 text-xs font-medium uppercase tracking-wide text-ink-faint">
              {t("login.email")}
            </dt>
            <dd class="min-w-0 text-right text-ink">{currentUser()?.email ?? ""}</dd>
          </div>
        </dl>
      </section>

      <div class="flex justify-end">
        <Button variant="secondary" icon="logout" onClick={signOut}>
          {t("nav.signOut")}
        </Button>
      </div>
    </div>
  );
}

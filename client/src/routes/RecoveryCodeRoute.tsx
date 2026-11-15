// Post-setup page explaining the recovery code file that was just downloaded.
import { useNavigate } from "@solidjs/router";
import { Button, Icon } from "../components/ui";
import { t } from "../lib/i18n";
import { cycleTheme, theme } from "../lib/theme";

export function RecoveryCodeRoute() {
  const navigate = useNavigate();

  return (
    <div class="flex min-h-screen flex-col bg-bg text-ink">
      <div class="flex items-center justify-end p-4">
        <button
          type="button"
          onClick={cycleTheme}
          class="rounded-md p-2 text-ink-muted hover:bg-surface-2 hover:text-ink"
          title={`${t("nav.theme")}: ${t(`theme.${theme()}`)}`}
        >
          <Icon
            name={theme() === "light" ? "sun" : theme() === "dark" ? "moon" : "computer-desktop"}
            size={18}
          />
        </button>
      </div>

      <div class="flex flex-1 items-center justify-center px-6 pb-16">
        <div class="w-full max-w-sm">
          <div class="mb-8 flex flex-col items-center text-center">
            <img src="/logo.svg" alt="Yily" class="h-16 w-auto" />
            <h1 class="mt-4 text-2xl font-semibold tracking-tight">{t("recoveryCode.title")}</h1>
          </div>

          <div class="space-y-4 rounded-xl border border-line bg-surface p-6 text-center">
            <div class="flex justify-center">
              <Icon name="key" size={26} class="text-ink-muted" />
            </div>
            <p class="text-sm text-ink-muted">{t("recoveryCode.body")}</p>
            <Button onClick={() => navigate("/login")} class="w-full">
              {t("recoveryCode.signIn")}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

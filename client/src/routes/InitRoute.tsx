// First-run setup page: create the initial admin account when there is none yet.
import { createSignal, Show } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { Button, Icon, Input } from "../components/ui";
import { Field } from "../components/shared";
import { t } from "../lib/i18n";
import { cycleTheme, theme } from "../lib/theme";
import { setup } from "../store";

export function InitRoute() {
  const navigate = useNavigate();
  const [name, setName] = createSignal("");
  const [email, setEmail] = createSignal("");
  const [password, setPassword] = createSignal("");
  const [confirm, setConfirm] = createSignal("");
  const [error, setError] = createSignal<string | null>(null);
  const [loading, setLoading] = createSignal(false);

  const submit = async (e: Event) => {
    e.preventDefault();
    if (!name().trim() || !email().trim().includes("@") || !password()) {
      setError(t("login.error"));
      return;
    }
    if (password() !== confirm()) {
      setError(t("settings.passwordMismatch"));
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await setup(name().trim(), email().trim(), password());
      navigate("/login", { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : t("login.error"));
    } finally {
      setLoading(false);
    }
  };

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
            <h1 class="mt-4 text-2xl font-semibold tracking-tight">{t("init.title")}</h1>
            <p class="mt-1 text-sm text-ink-muted">{t("init.subtitle")}</p>
          </div>

          <form onSubmit={submit} class="space-y-4 rounded-xl border border-line bg-surface p-6">
            <Field label={t("common.name")}>
              <Input
                value={name()}
                onInput={(e) => setName(e.currentTarget.value)}
                autocomplete="name"
              />
            </Field>

            <Field label={t("login.email")}>
              <Input
                type="email"
                value={email()}
                onInput={(e) => setEmail(e.currentTarget.value)}
                placeholder={t("login.emailPlaceholder")}
                autocomplete="email"
              />
            </Field>

            <Field label={t("login.password")}>
              <Input
                type="password"
                value={password()}
                onInput={(e) => setPassword(e.currentTarget.value)}
                placeholder={t("login.passwordPlaceholder")}
                autocomplete="new-password"
              />
            </Field>

            <Field label={t("settings.confirmPassword")}>
              <Input
                type="password"
                value={confirm()}
                onInput={(e) => setConfirm(e.currentTarget.value)}
                autocomplete="new-password"
              />
            </Field>

            <Show when={error()}>
              <p class="text-sm text-danger">{error()}</p>
            </Show>

            <Button type="submit" loading={loading()} class="w-full">
              {t("init.submit")}
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}

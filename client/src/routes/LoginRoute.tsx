// Mock login page (email + password) that routes into the app on submit.
import { createSignal, Show } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { Button, Icon, Input } from "../components/ui";
import { Field } from "../components/shared";
import { t } from "../lib/i18n";
import { cycleTheme, theme } from "../lib/theme";

export function LoginRoute() {
  const navigate = useNavigate();
  const [email, setEmail] = createSignal("");
  const [password, setPassword] = createSignal("");
  const [show, setShow] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [loading, setLoading] = createSignal(false);

  const submit = (e: Event) => {
    e.preventDefault();
    const em = email().trim();
    if (!em.includes("@") || !password()) {
      setError(t("login.error"));
      return;
    }
    setError(null);
    setLoading(true);
    // Mock auth — no backend yet, so route into the app after a beat.
    setTimeout(() => navigate("/projects"), 400);
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
          <Icon name={theme() === "light" ? "sun" : theme() === "dark" ? "moon" : "computer-desktop"} size={18} />
        </button>
      </div>

      <div class="flex flex-1 items-center justify-center px-6 pb-16">
        <div class="w-full max-w-sm">
          <div class="mb-8 flex flex-col items-center text-center">
            <img src="/logo.svg" alt="Yily" class="h-16 w-auto" />
            <h1 class="mt-4 text-2xl font-semibold tracking-tight">{t("login.title")}</h1>
            <p class="mt-1 text-sm text-ink-muted">{t("login.subtitle")}</p>
          </div>

          <form onSubmit={submit} class="space-y-4 rounded-xl border border-line bg-surface p-6">
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
              <div class="relative">
                <Input
                  type={show() ? "text" : "password"}
                  value={password()}
                  onInput={(e) => setPassword(e.currentTarget.value)}
                  placeholder={t("login.passwordPlaceholder")}
                  autocomplete="current-password"
                  style={{ "padding-right": "2.5rem" }}
                />
                <button
                  type="button"
                  onClick={() => setShow((s) => !s)}
                  class="absolute inset-y-0 right-0 flex items-center px-3 text-ink-faint hover:text-ink"
                  aria-label={show() ? t("login.hidePassword") : t("login.showPassword")}
                >
                  <Icon name={show() ? "eye-off" : "eye"} size={16} />
                </button>
              </div>
            </Field>

            <Show when={error()}>
              <p class="text-sm text-danger">{error()}</p>
            </Show>

            <Button type="submit" loading={loading()} class="w-full">
              {t("login.submit")}
            </Button>

            <div class="text-center">
              <button type="button" class="text-sm text-brand hover:underline">
                {t("login.forgot")}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}

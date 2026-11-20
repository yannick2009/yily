import { createSignal, Show } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { Button, Icon, Input } from "../components/ui";
import { Field } from "../components/shared";
import { ApiError } from "../api/client";
import { t } from "../lib/i18n";
import { errorMessage } from "../lib/errors";
import { cycleTheme, theme } from "../lib/theme";
import { login, loginWithTOTP } from "../store";

// Keep only digits so pasted codes with spaces or dashes are accepted.
function normalizeCode(value: string): string {
  return value.replace(/\D/g, "").slice(0, 6);
}

// LoginRoute displays the login form and handles password, TOTP second-factor
// and passwordless authenticator-code authentication.
export function LoginRoute() {
  const navigate = useNavigate();
  const [email, setEmail] = createSignal("");
  const [password, setPassword] = createSignal("");
  const [code, setCode] = createSignal("");
  const [mode, setMode] = createSignal<"password" | "totp">("password");
  const [totpRequired, setTotpRequired] = createSignal(false);
  const [show, setShow] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [loading, setLoading] = createSignal(false);

  // The code field appears either in passwordless mode or once the server has
  // confirmed that the password was correct and a second factor is required.
  const codeVisible = () => mode() === "totp" || totpRequired();

  // Switching between password and passwordless login resets the transient
  // second-factor state so a stale code is never submitted.
  const switchMode = () => {
    setMode((current) => (current === "password" ? "totp" : "password"));
    setTotpRequired(false);
    setCode("");
    setError(null);
  };

  const submit = async (e: Event) => {
    e.preventDefault();
    const em = email().trim();

    if (!em.includes("@")) {
      setError(t("login.error"));
      return;
    }
    if (mode() === "password" && !password()) {
      setError(t("login.error"));
      return;
    }
    if (codeVisible() && code().length !== 6) {
      setError(t("login.errorTotp"));
      return;
    }

    setError(null);
    setLoading(true);
    try {
      if (mode() === "totp") {
        await loginWithTOTP(em, code());
      } else {
        await login(em, password(), totpRequired() ? code() : undefined);
      }
      navigate("/", { replace: true });
    } catch (err) {
      if (err instanceof ApiError && err.code === "AUTH_TOTP_REQUIRED") {
        // The password was correct: reveal the second-factor field.
        setTotpRequired(true);
        setError(null);
      } else {
        if (err instanceof ApiError && err.code === "AUTH_INVALID_TOTP_CODE") {
          setTotpRequired(true);
        }
        setError(errorMessage(err));
      }
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

            <Show when={mode() === "password"}>
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
            </Show>

            <Show when={codeVisible()}>
              <Field label={t("login.totpCode")}>
                <Input
                  value={code()}
                  onInput={(e) => setCode(normalizeCode(e.currentTarget.value))}
                  placeholder={t("login.totpPlaceholder")}
                  inputMode="numeric"
                  pattern="[0-9]*"
                  maxlength={6}
                  autocomplete="one-time-code"
                />
              </Field>
              <p class="-mt-2 text-xs text-ink-faint">{t("login.totpRequired")}</p>
            </Show>

            <Show when={error()}>
              <p class="text-sm text-danger">{error()}</p>
            </Show>

            <Button type="submit" loading={loading()} class="w-full">
              {t("login.submit")}
            </Button>

            <div class="flex flex-col items-center gap-2 text-center">
              <button
                type="button"
                class="text-sm text-brand hover:underline"
                onClick={switchMode}
              >
                {mode() === "password" ? t("login.useTotp") : t("login.usePassword")}
              </button>
              <button
                type="button"
                class="text-sm text-brand hover:underline"
                onClick={() => navigate("/recover")}
              >
                {t("login.forgot")}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}

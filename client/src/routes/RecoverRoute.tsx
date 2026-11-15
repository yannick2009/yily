// Password recovery page: reset the Super Admin password using the one-time
// recovery code downloaded at account creation.
import { createSignal, Show } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { Button, Icon, Input } from "../components/ui";
import { Field } from "../components/shared";
import { t } from "../lib/i18n";
import { errorMessage } from "../lib/errors";
import { isStrongPassword } from "../lib/password";
import { cycleTheme, theme } from "../lib/theme";
import { recover } from "../api/auth";

export function RecoverRoute() {
  const navigate = useNavigate();
  const [recoveryCode, setRecoveryCode] = createSignal("");
  const [newPassword, setNewPassword] = createSignal("");
  const [confirm, setConfirm] = createSignal("");
  const [show, setShow] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [loading, setLoading] = createSignal(false);
  const [done, setDone] = createSignal(false);

  const submit = async (e: Event) => {
    e.preventDefault();
    if (!recoveryCode().trim()) {
      setError(t("recover.codeRequired"));
      return;
    }
    if (newPassword().length < 12) {
      setError(t("init.passwordMin"));
      return;
    }
    if (newPassword().length > 128) {
      setError(t("init.passwordMax"));
      return;
    }
    if (!isStrongPassword(newPassword())) {
      setError(t("init.passwordComplexity"));
      return;
    }
    if (newPassword() !== confirm()) {
      setError(t("settings.passwordMismatch"));
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await recover({
        recovery_code: recoveryCode().trim(),
        password: newPassword(),
        confirm: confirm(),
      });
      setRecoveryCode("");
      setNewPassword("");
      setConfirm("");
      setDone(true);
    } catch (err) {
      setError(errorMessage(err));
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
            <h1 class="mt-4 text-2xl font-semibold tracking-tight">{t("recover.title")}</h1>
            <p class="mt-1 text-sm text-ink-muted">{t("recover.subtitle")}</p>
          </div>

          <Show
            when={done()}
            fallback={
              <form
                onSubmit={submit}
                class="space-y-4 rounded-xl border border-line bg-surface p-6"
              >
                <p class="text-sm text-ink-muted">{t("recover.hint")}</p>

                <Field label={t("recover.code")}>
                  <Input
                    value={recoveryCode()}
                    onInput={(e) => setRecoveryCode(e.currentTarget.value)}
                    placeholder={t("recover.codePlaceholder")}
                    autocomplete="off"
                  />
                </Field>

                <Field label={t("login.password")}>
                  <div class="relative">
                    <Input
                      type={show() ? "text" : "password"}
                      value={newPassword()}
                      onInput={(e) => setNewPassword(e.currentTarget.value)}
                      placeholder={t("login.passwordPlaceholder")}
                      autocomplete="new-password"
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
                  {t("recover.submit")}
                </Button>

                <div class="text-center">
                  <button
                    type="button"
                    class="text-sm text-brand hover:underline"
                    onClick={() => navigate("/login")}
                  >
                    {t("recover.back")}
                  </button>
                </div>
              </form>
            }
          >
            <div class="space-y-4 rounded-xl border border-line bg-surface p-6 text-center">
              <p class="text-sm text-success">{t("recover.success")}</p>
              <Button onClick={() => navigate("/login")} class="w-full">
                {t("recover.back")}
              </Button>
            </div>
          </Show>
        </div>
      </div>
    </div>
  );
}

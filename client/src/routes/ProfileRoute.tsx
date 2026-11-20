// Dedicated profile page: account identity, two-factor authentication and sign-out.
import { createSignal, Show } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { auth } from "../api";
import type { TOTPSetup } from "../api/auth";
import { Button, Chip, Icon, Input } from "../components/ui";
import { Dialog, Field } from "../components/shared";
import { copyToClipboard } from "../lib/clipboard";
import { t } from "../lib/i18n";
import { errorMessage } from "../lib/errors";
import { currentUser, logout, setTOTPEnabled } from "../store";

// Keep only digits so pasted codes with spaces or dashes are accepted.
function normalizeCode(value: string): string {
  return value.replace(/\D/g, "").slice(0, 6);
}

export function ProfileRoute() {
  const navigate = useNavigate();
  const [setupOpen, setSetupOpen] = createSignal(false);
  const [setup, setSetup] = createSignal<TOTPSetup | null>(null);
  const [code, setCode] = createSignal("");
  const [disableOpen, setDisableOpen] = createSignal(false);
  const [disablePassword, setDisablePassword] = createSignal("");
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [notice, setNotice] = createSignal<string | null>(null);
  const [copied, setCopied] = createSignal(false);

  const totpEnabled = () => currentUser()?.totp_enabled === true;

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

  const startSetup = async () => {
    setError(null);
    setNotice(null);
    setBusy(true);
    try {
      const result = await auth.totpSetup();
      setSetup(result);
      setCode("");
      setSetupOpen(true);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const closeSetup = () => {
    setSetupOpen(false);
    setSetup(null);
    setCode("");
    setError(null);
  };

  const confirmSetup = async () => {
    if (code().length !== 6) {
      setError(t("settings.totpCodeInvalid"));
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await auth.totpConfirm(code());
      setTOTPEnabled(true);
      closeSetup();
      setNotice(t("settings.totpEnabledNotice"));
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const openDisable = () => {
    setError(null);
    setNotice(null);
    setDisablePassword("");
    setDisableOpen(true);
  };

  const closeDisable = () => {
    setDisableOpen(false);
    setDisablePassword("");
    setError(null);
  };

  const disable = async () => {
    setBusy(true);
    setError(null);
    try {
      await auth.totpDisable(disablePassword());
      setTOTPEnabled(false);
      closeDisable();
      setNotice(t("settings.totpDisabledNotice"));
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const copySecret = async () => {
    const current = setup();
    if (!current) return;
    if (await copyToClipboard(current.secret)) {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    }
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

      <section class="rounded-xl border border-line bg-surface p-4">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <h2 class="text-sm font-semibold text-ink">{t("settings.totpTitle")}</h2>
            <p class="mt-1 text-xs text-ink-muted">{t("settings.totpDescription")}</p>
          </div>
          <Chip variant={totpEnabled() ? "success" : "neutral"}>
            {totpEnabled() ? t("settings.totpEnabled") : t("settings.totpDisabled")}
          </Chip>
        </div>

        <div class="mt-3">
          <Button
            variant={totpEnabled() ? "secondary" : "primary"}
            icon={totpEnabled() ? "key-off" : "lock"}
            loading={busy() && !setupOpen() && !disableOpen()}
            onClick={totpEnabled() ? openDisable : startSetup}
          >
            {totpEnabled() ? t("settings.totpDisable") : t("settings.totpEnable")}
          </Button>
        </div>

        <Show when={notice()}>
          <p class="mt-3 text-sm text-success">{notice()}</p>
        </Show>
        <Show when={error() && !setupOpen() && !disableOpen()}>
          <p class="mt-3 text-sm text-danger">{error()}</p>
        </Show>
      </section>

      <div class="flex justify-end">
        <Button variant="secondary" icon="logout" onClick={signOut}>
          {t("nav.signOut")}
        </Button>
      </div>

      <Dialog
        open={setupOpen()}
        onClose={closeSetup}
        title={t("settings.totpSetupTitle")}
        footer={
          <>
            <Button variant="secondary" onClick={closeSetup}>
              {t("common.cancel")}
            </Button>
            <Button onClick={confirmSetup} loading={busy()} disabled={code().length !== 6}>
              {t("settings.totpConfirm")}
            </Button>
          </>
        }
      >
        <p class="text-sm text-ink-muted">{t("settings.totpScan")}</p>

        <Show when={setup()?.qr_png}>
          <div class="mt-4 flex justify-center">
            <img
              src={setup()!.qr_png}
              alt="TOTP QR code"
              class="h-48 w-48 rounded-lg bg-white p-2"
            />
          </div>
        </Show>

        <div class="mt-4 space-y-1.5">
          <span class="block text-sm font-medium text-ink">{t("settings.totpSecret")}</span>
          <div class="flex items-center gap-2">
            <code class="min-w-0 flex-1 break-all rounded-md bg-surface-2 px-2 py-1 font-mono text-xs text-ink">
              {setup()?.secret}
            </code>
            <button
              type="button"
              onClick={copySecret}
              class="shrink-0 rounded-md p-2 text-ink-muted hover:bg-surface-2 hover:text-ink"
              title={t("common.copy")}
              aria-label={t("common.copy")}
            >
              <Icon name={copied() ? "check" : "copy"} size={16} />
            </button>
          </div>
          <p class="text-xs text-ink-faint">{t("settings.totpSecretHint")}</p>
        </div>

        <div class="mt-4">
          <Field label={t("settings.totpCode")}>
            <Input
              value={code()}
              onInput={(e) => setCode(normalizeCode(e.currentTarget.value))}
              inputMode="numeric"
              pattern="[0-9]*"
              maxlength={6}
              autocomplete="one-time-code"
              placeholder="123456"
            />
          </Field>
        </div>

        <Show when={error()}>
          <p class="mt-3 text-sm text-danger">{error()}</p>
        </Show>
      </Dialog>

      <Dialog
        open={disableOpen()}
        onClose={closeDisable}
        title={t("settings.totpDisableTitle")}
        footer={
          <>
            <Button variant="secondary" onClick={closeDisable}>
              {t("common.cancel")}
            </Button>
            <Button variant="danger" onClick={disable} loading={busy()} disabled={!disablePassword()}>
              {t("settings.totpDisable")}
            </Button>
          </>
        }
      >
        <p class="text-sm text-ink-muted">{t("settings.totpDisableHint")}</p>

        <div class="mt-4">
          <Field label={t("login.password")}>
            <Input
              type="password"
              value={disablePassword()}
              onInput={(e) => setDisablePassword(e.currentTarget.value)}
              autocomplete="current-password"
            />
          </Field>
        </div>

        <Show when={error()}>
          <p class="mt-3 text-sm text-danger">{error()}</p>
        </Show>
      </Dialog>
    </div>
  );
}

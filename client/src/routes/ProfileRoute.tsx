// Dedicated profile page: account identity and editing, password, two-factor authentication and sign-out.
import { createSignal, Show } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { auth } from "../api";
import type { TOTPSetup } from "../types";
import { Button, Chip, Icon, Input } from "../components/ui";
import { Dialog, Field } from "../components/shared";
import { copyToClipboard } from "../lib/clipboard";
import { t } from "../lib/i18n";
import { errorMessage } from "../lib/errors";
import { isStrongPassword } from "../lib/password";
import { currentUser, logout, setCurrentUser, setTOTPEnabled } from "../store";

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
  const [profileOpen, setProfileOpen] = createSignal(false);
  const [profileName, setProfileName] = createSignal("");
  const [profileEmail, setProfileEmail] = createSignal("");
  const [profilePassword, setProfilePassword] = createSignal("");
  const [profileBusy, setProfileBusy] = createSignal(false);
  const [profileError, setProfileError] = createSignal<string | null>(null);
  const [profileNotice, setProfileNotice] = createSignal<string | null>(null);
  const [passwordOpen, setPasswordOpen] = createSignal(false);
  const [currentPassword, setCurrentPassword] = createSignal("");
  const [newPassword, setNewPassword] = createSignal("");
  const [confirmPassword, setConfirmPassword] = createSignal("");
  const [showNewPassword, setShowNewPassword] = createSignal(false);
  const [passwordBusy, setPasswordBusy] = createSignal(false);
  const [passwordError, setPasswordError] = createSignal<string | null>(null);
  const [passwordNotice, setPasswordNotice] = createSignal<string | null>(null);

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

  const openProfile = () => {
    setProfileName(currentUser()?.name ?? "");
    setProfileEmail(currentUser()?.email ?? "");
    setProfilePassword("");
    setProfileError(null);
    setProfileOpen(true);
  };

  const closeProfile = () => {
    setProfileOpen(false);
    setProfilePassword("");
    setProfileError(null);
  };

  const emailChanged = () => profileEmail().trim().toLowerCase() !== (currentUser()?.email ?? "");

  const saveProfile = async (event: Event) => {
    event.preventDefault();
    const name = profileName().trim();
    const email = profileEmail().trim();
    if (name === "") {
      setProfileError(t("settings.nameRequired"));
      return;
    }
    if (!email.includes("@")) {
      setProfileError(t("settings.emailInvalid"));
      return;
    }
    const changingEmail = emailChanged();
    if (changingEmail && profilePassword() === "") {
      setProfileError(t("settings.currentPasswordRequired"));
      return;
    }

    setProfileBusy(true);
    setProfileError(null);
    try {
      const updated = await auth.updateProfile({
        name,
        email,
        ...(changingEmail ? { current_password: profilePassword() } : {}),
      });
      setCurrentUser(updated);
      setProfileOpen(false);
      setProfilePassword("");
      setProfileNotice(t("settings.profileUpdated"));
    } catch (err) {
      setProfileError(errorMessage(err));
    } finally {
      setProfileBusy(false);
    }
  };

  const openPassword = () => {
    setCurrentPassword("");
    setNewPassword("");
    setConfirmPassword("");
    setShowNewPassword(false);
    setPasswordError(null);
    setPasswordNotice(null);
    setPasswordOpen(true);
  };

  const closePassword = () => {
    setPasswordOpen(false);
    setCurrentPassword("");
    setNewPassword("");
    setConfirmPassword("");
    setPasswordError(null);
  };

  const savePassword = async (event: Event) => {
    event.preventDefault();
    if (currentPassword() === "") {
      setPasswordError(t("settings.enterCurrentPassword"));
      return;
    }
    if (newPassword().length < 12) {
      setPasswordError(t("init.passwordMin"));
      return;
    }
    if (newPassword().length > 128) {
      setPasswordError(t("init.passwordMax"));
      return;
    }
    if (!isStrongPassword(newPassword())) {
      setPasswordError(t("init.passwordComplexity"));
      return;
    }
    if (newPassword() !== confirmPassword()) {
      setPasswordError(t("settings.passwordMismatch"));
      return;
    }

    setPasswordBusy(true);
    setPasswordError(null);
    try {
      await auth.changePassword({
        current_password: currentPassword(),
        new_password: newPassword(),
        confirm: confirmPassword(),
      });
      closePassword();
      setPasswordNotice(t("settings.passwordChanged"));
    } catch (err) {
      setPasswordError(errorMessage(err));
    } finally {
      setPasswordBusy(false);
    }
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
        <div class="flex items-start justify-between gap-4">
          <h2 class="text-sm font-semibold text-ink">{t("settings.profile")}</h2>
          <Button variant="secondary" icon="pencil" onClick={openProfile}>
            {t("settings.profileEdit")}
          </Button>
        </div>
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
        <Show when={profileNotice()}>
          <p class="mt-3 text-sm text-success">{profileNotice()}</p>
        </Show>
      </section>

      <section class="rounded-xl border border-line bg-surface p-4">
        <div class="flex items-start justify-between gap-4">
          <div class="min-w-0">
            <h2 class="text-sm font-semibold text-ink">{t("settings.password")}</h2>
            <p class="mt-1 text-xs text-ink-muted">{t("settings.passwordDescription")}</p>
          </div>
          <Button variant="secondary" icon="lock" onClick={openPassword}>
            {t("settings.changePassword")}
          </Button>
        </div>
        <Show when={passwordNotice()}>
          <p class="mt-3 text-sm text-success">{passwordNotice()}</p>
        </Show>
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
            <Button
              variant="danger"
              onClick={disable}
              loading={busy()}
              disabled={!disablePassword()}
            >
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

      <Dialog
        open={profileOpen()}
        onClose={closeProfile}
        title={t("settings.profileEdit")}
        footer={
          <>
            <Button variant="secondary" onClick={closeProfile}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" form="profile-form" loading={profileBusy()}>
              {t("common.save")}
            </Button>
          </>
        }
      >
        <form id="profile-form" onSubmit={saveProfile} class="space-y-4">
          <Field label={t("common.name")}>
            <Input
              value={profileName()}
              onInput={(e) => setProfileName(e.currentTarget.value)}
              autocomplete="name"
            />
          </Field>
          <Field label={t("login.email")}>
            <Input
              type="email"
              value={profileEmail()}
              onInput={(e) => setProfileEmail(e.currentTarget.value)}
              autocomplete="email"
            />
          </Field>
          <Show when={emailChanged()}>
            <Field
              label={t("settings.currentPassword")}
              hint={t("settings.currentPasswordRequired")}
            >
              <Input
                type="password"
                value={profilePassword()}
                onInput={(e) => setProfilePassword(e.currentTarget.value)}
                autocomplete="current-password"
              />
            </Field>
          </Show>
          <Show when={profileError()}>
            <p class="text-sm text-danger">{profileError()}</p>
          </Show>
        </form>
      </Dialog>

      <Dialog
        open={passwordOpen()}
        onClose={closePassword}
        title={t("settings.changePassword")}
        footer={
          <>
            <Button variant="secondary" onClick={closePassword}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" form="password-form" loading={passwordBusy()}>
              {t("common.save")}
            </Button>
          </>
        }
      >
        <form id="password-form" onSubmit={savePassword} class="space-y-4">
          <Field label={t("settings.currentPassword")}>
            <Input
              type="password"
              value={currentPassword()}
              onInput={(e) => setCurrentPassword(e.currentTarget.value)}
              autocomplete="current-password"
            />
          </Field>
          <Field label={t("settings.newPassword")}>
            <div class="relative">
              <Input
                type={showNewPassword() ? "text" : "password"}
                value={newPassword()}
                onInput={(e) => setNewPassword(e.currentTarget.value)}
                autocomplete="new-password"
                style={{ "padding-right": "2.5rem" }}
              />
              <button
                type="button"
                onClick={() => setShowNewPassword((visible) => !visible)}
                class="absolute inset-y-0 right-0 flex items-center px-3 text-ink-faint hover:text-ink"
                aria-label={showNewPassword() ? t("login.hidePassword") : t("login.showPassword")}
              >
                <Icon name={showNewPassword() ? "eye-off" : "eye"} size={16} />
              </button>
            </div>
          </Field>
          <Field label={t("settings.confirmPassword")}>
            <Input
              type={showNewPassword() ? "text" : "password"}
              value={confirmPassword()}
              onInput={(e) => setConfirmPassword(e.currentTarget.value)}
              autocomplete="new-password"
            />
          </Field>
          <Show when={passwordError()}>
            <p class="text-sm text-danger">{passwordError()}</p>
          </Show>
        </form>
      </Dialog>
    </div>
  );
}

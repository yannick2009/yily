// Dedicated profile page: account identity (name) and password change.
import { createSignal, Show } from "solid-js";
import { A } from "@solidjs/router";
import { Button, Icon, Input } from "../components/ui";
import { Field } from "../components/shared";
import { t } from "../lib/i18n";
import { currentUser, updateCurrentUser, updatePassword } from "../store";

export function ProfileRoute() {
  const [name, setName] = createSignal(currentUser().name);

  const [currentPw, setCurrentPw] = createSignal("");
  const [newPw, setNewPw] = createSignal("");
  const [confirmPw, setConfirmPw] = createSignal("");
  const [passwordError, setPasswordError] = createSignal<string | null>(null);
  const [passwordMessage, setPasswordMessage] = createSignal<string | null>(null);

  const initials = () =>
    currentUser()
      .name.split(" ")
      .map((part) => part[0] ?? "")
      .join("")
      .toUpperCase();

  const saveProfile = () => updateCurrentUser({ name: name().trim() });

  const changePassword = () => {
    setPasswordError(null);
    setPasswordMessage(null);
    if (newPw() !== confirmPw()) {
      setPasswordError(t("settings.passwordMismatch"));
      return;
    }
    if (!updatePassword(currentPw(), newPw())) {
      setPasswordError(t("settings.passwordWrong"));
      return;
    }
    setCurrentPw("");
    setNewPw("");
    setConfirmPw("");
    setPasswordMessage(t("settings.passwordChanged"));
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
          <h1 class="truncate text-xl font-semibold text-ink">{currentUser().name}</h1>
          <p class="truncate text-sm text-ink-muted">{currentUser().email}</p>
        </div>
      </div>

      <section class="rounded-xl border border-line bg-surface p-4">
        <h2 class="text-sm font-semibold text-ink">{t("settings.profile")}</h2>
        <form
          class="mt-4 space-y-3"
          onSubmit={(e) => {
            e.preventDefault();
            saveProfile();
          }}
        >
          <Field label={t("common.name")}>
            <Input
              value={name()}
              onInput={(e) => setName(e.currentTarget.value)}
              autocomplete="name"
            />
          </Field>
          <Field label={t("login.email")}>
            <p class="text-sm text-ink-muted">{currentUser().email}</p>
          </Field>
          <Button type="submit">{t("common.save")}</Button>
        </form>
      </section>

      <section class="rounded-xl border border-line bg-surface p-4">
        <h2 class="text-sm font-semibold text-ink">{t("settings.password")}</h2>
        <form
          class="mt-4 space-y-3"
          onSubmit={(e) => {
            e.preventDefault();
            changePassword();
          }}
        >
          <Field label={t("settings.currentPassword")}>
            <Input
              type="password"
              value={currentPw()}
              onInput={(e) => setCurrentPw(e.currentTarget.value)}
              autocomplete="current-password"
            />
          </Field>
          <Field label={t("settings.newPassword")}>
            <Input
              type="password"
              value={newPw()}
              onInput={(e) => setNewPw(e.currentTarget.value)}
              autocomplete="new-password"
            />
          </Field>
          <Field label={t("settings.confirmPassword")}>
            <Input
              type="password"
              value={confirmPw()}
              onInput={(e) => setConfirmPw(e.currentTarget.value)}
              autocomplete="new-password"
            />
          </Field>
          <Show when={passwordError()}>
            <p class="text-sm text-danger">{passwordError()}</p>
          </Show>
          <Show when={passwordMessage()}>
            <p class="text-sm text-success">{passwordMessage()}</p>
          </Show>
          <Button type="submit">{t("common.save")}</Button>
        </form>
      </section>

      <div class="flex justify-end">
        <A
          href="/login"
          class="inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium text-ink-muted hover:bg-surface-2 hover:text-ink"
        >
          <Icon name="logout" size={15} />
          {t("nav.signOut")}
        </A>
      </div>
    </div>
  );
}

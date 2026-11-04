// Rotate-token dialog. Confirms the rotation then shows the new secret once (one-time reveal).
import { createEffect, createSignal, Show } from "solid-js";
import { tokens } from "../../api";
import { t } from "../../lib/i18n";
import { copyToClipboard } from "../../lib/clipboard";
import type { Token } from "../../types";
import { Button, Icon } from "../ui";
import { Dialog, Field } from "../shared";
import { ExpirationPicker } from "./ExpirationPicker";
import { DEFAULT_EXPIRATION_DAYS, EXPIRATION_NEVER } from "./constants";

export function RotateTokenDialog(props: {
  token: Token | null;
  onClose: () => void;
  onRotated: () => unknown;
}) {
  const [days, setDays] = createSignal<number>(DEFAULT_EXPIRATION_DAYS);
  const [rotated, setRotated] = createSignal<string | null>(null);
  const [copied, setCopied] = createSignal(false);
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  // Reset local state whenever the dialog opens with a new target token.
  createEffect(() => {
    if (props.token) {
      setDays(DEFAULT_EXPIRATION_DAYS);
      setRotated(null);
      setCopied(false);
      setError(null);
    }
  });

  const submit = async (e: Event) => {
    e.preventDefault();
    const target = props.token;
    if (!target) return;
    setBusy(true);
    setError(null);
    try {
      const d = days();
      const newToken = await tokens.rotate(target.id, {
        expires_in_days: d === EXPIRATION_NEVER ? null : d,
      });
      setRotated(newToken.token);
      await props.onRotated();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.save"));
    } finally {
      setBusy(false);
    }
  };

  const copy = async () => {
    const s = rotated();
    if (!s) return;
    if (await copyToClipboard(s)) setCopied(true);
  };

  return (
    <Dialog
      open={!!props.token}
      onClose={props.onClose}
      title={rotated() === null ? t("tokens.rotateTitle") : t("tokens.rotatedTitle")}
      footer={
        rotated() === null ? (
          <>
            <Button variant="secondary" onClick={props.onClose}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" form="rotate-token-form" loading={busy()}>
              {t("tokens.rotate")}
            </Button>
          </>
        ) : (
          <Button variant="secondary" onClick={props.onClose}>
            {t("tokens.done")}
          </Button>
        )
      }
    >
      <Show
        when={rotated() === null}
        fallback={
          <div class="space-y-4">
            <div class="rounded-lg border border-line bg-surface-2 p-3">
              <p class="break-all font-mono text-sm text-ink">{rotated()}</p>
            </div>
            <div class="flex items-start gap-2 rounded-lg bg-brand-subtle p-3 text-sm text-brand">
              <Icon name="lock" size={16} class="mt-0.5 shrink-0" />
              <p>{t("tokens.copyNow")}</p>
            </div>
            <Button
              variant="secondary"
              icon={copied() ? "check" : "copy"}
              onClick={copy}
              class="w-full"
            >
              {copied() ? t("tokens.copied") : t("tokens.copy")}
            </Button>
          </div>
        }
      >
        <form id="rotate-token-form" onSubmit={submit} class="space-y-4">
          <p class="text-sm text-ink-muted">
            {t("tokens.rotateConfirm", { name: props.token?.name ?? "" })}
          </p>
          <Field label={t("tokens.expirationLabel")} hint={t("tokens.expirationHint")}>
            <ExpirationPicker
              value={days()}
              onChange={setDays}
              allowNever={props.token?.scope === "read"}
            />
          </Field>
          <Show when={error()}>
            <p class="text-sm text-danger">{error()}</p>
          </Show>
        </form>
      </Show>
    </Dialog>
  );
}

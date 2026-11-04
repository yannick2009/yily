// Revoke-token confirmation dialog.
import { createEffect, createSignal, Show } from "solid-js";
import { tokens } from "../../api";
import { t } from "../../lib/i18n";
import { errorMessage } from "../../lib/errors";
import type { Token } from "../../types";
import { Button } from "../ui";
import { Dialog } from "../shared";

export function RevokeTokenDialog(props: {
  token: Token | null;
  onClose: () => void;
  onRevoked: () => unknown;
}) {
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  createEffect(() => {
    if (props.token) setError(null);
  });

  const confirm = async () => {
    const target = props.token;
    if (!target) return;
    setBusy(true);
    setError(null);
    try {
      await tokens.revoke(target.id);
      await props.onRevoked();
      props.onClose();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog
      open={!!props.token}
      onClose={props.onClose}
      title={t("tokens.revokeTitle")}
      footer={
        <>
          <Button variant="secondary" onClick={props.onClose}>
            {t("common.cancel")}
          </Button>
          <Button variant="danger" onClick={confirm} loading={busy()}>
            {t("tokens.revoke")}
          </Button>
        </>
      }
    >
      <p class="text-sm text-ink-muted">
        {t("tokens.revokeConfirm", { name: props.token?.name ?? "" })}
      </p>
      <Show when={error()}>
        <p class="mt-3 text-sm text-danger">{error()}</p>
      </Show>
    </Dialog>
  );
}

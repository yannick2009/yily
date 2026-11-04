// Confirm variable deletion.
import { createEffect, createSignal, Show } from "solid-js";
import { variables } from "../../api";
import { t } from "../../lib/i18n";
import { errorMessage } from "../../lib/errors";
import { Button } from "../ui";
import { Dialog } from "../shared";

export function DeleteVariableDialog(props: {
  target: { id: string; name: string } | null;
  onClose: () => void;
  onDeleted: () => unknown;
}) {
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  createEffect(() => {
    if (props.target) setError(null);
  });

  const confirm = async () => {
    const d = props.target;
    if (!d) return;
    setBusy(true);
    setError(null);
    try {
      await variables.remove(d.id);
      await props.onDeleted();
      props.onClose();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog
      open={!!props.target}
      onClose={props.onClose}
      title={t("env.deleteVariable")}
      footer={
        <>
          <Button variant="secondary" onClick={props.onClose}>
            {t("common.cancel")}
          </Button>
          <Button variant="danger" onClick={confirm} loading={busy()}>
            {t("common.delete")}
          </Button>
        </>
      }
    >
      <p class="text-sm text-ink-muted">
        {t("env.deleteVariableConfirm", { name: props.target?.name ?? "" })}
      </p>
      <Show when={error()}>
        <p class="mt-3 text-sm text-danger">{error()}</p>
      </Show>
    </Dialog>
  );
}

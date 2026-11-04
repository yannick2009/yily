// Create-variable dialog: name + value form, with a value generator.
import { createSignal, Show } from "solid-js";
import { variables } from "../../api";
import { t } from "../../lib/i18n";
import { errorMessage } from "../../lib/errors";
import { normalizeVariableName } from "../../lib/variable";
import { Button, Input, Textarea } from "../ui";
import { Dialog, Field, ValueGenerator } from "../shared";

export function CreateVariableDialog(props: {
  open: boolean;
  onClose: () => void;
  envId: string;
  onCreated: () => unknown;
}) {
  const [name, setName] = createSignal("");
  const [value, setValue] = createSignal("");
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  const close = () => {
    setName("");
    setValue("");
    setError(null);
    props.onClose();
  };

  const submit = async (e: Event) => {
    e.preventDefault();
    const n = normalizeVariableName(name());
    if (!n) return;
    setBusy(true);
    setError(null);
    try {
      await variables.create(props.envId, { name: n, value: value() });
      setName("");
      setValue("");
      await props.onCreated();
      props.onClose();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog
      open={props.open}
      onClose={close}
      title={t("env.newVariable")}
      footer={
        <>
          <Button variant="secondary" onClick={close}>
            {t("common.cancel")}
          </Button>
          <Button type="submit" form="new-variable-form" loading={busy()}>
            {t("env.createVariable")}
          </Button>
        </>
      }
    >
      <form id="new-variable-form" onSubmit={submit} class="space-y-4">
        <Field label={t("common.name")}>
          <Input
            value={name()}
            onInput={(e) => setName(normalizeVariableName(e.currentTarget.value))}
            placeholder={t("env.varNamePlaceholder")}
          />
        </Field>
        <Field label={t("env.value")} hint={t("env.encryptedHint")}>
          <Textarea
            rows={3}
            value={value()}
            onInput={(e) => setValue(e.currentTarget.value)}
            placeholder={t("env.varValuePlaceholder")}
          />
          <ValueGenerator onGenerate={(v) => setValue(v)} />
        </Field>
        <Show when={error()}>
          <p class="text-sm text-danger">{error()}</p>
        </Show>
      </form>
    </Dialog>
  );
}

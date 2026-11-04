// Import variables from an .env, .json or .yaml file. Drag-and-drop or click to browse.
import { createEffect, createSignal, Show } from "solid-js";
import { environments } from "../../api";
import { t } from "../../lib/i18n";
import { errorMessage } from "../../lib/errors";
import { Button, Icon } from "../ui";
import { Dialog } from "../shared";

export function ImportVariablesDialog(props: {
  open: boolean;
  onClose: () => void;
  envId: string;
  onImported: () => unknown;
}) {
  let fileInput: HTMLInputElement | undefined;
  const [dragging, setDragging] = createSignal(false);
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  // Reset the native input and transient state whenever the dialog opens.
  createEffect(() => {
    if (props.open) {
      queueMicrotask(() => {
        if (fileInput) fileInput.value = "";
      });
      setDragging(false);
      setError(null);
    }
  });

  const close = () => {
    setDragging(false);
    setError(null);
    props.onClose();
  };

  const handleFile = async (file: File) => {
    setBusy(true);
    setError(null);
    try {
      await environments.importFile(props.envId, file);
      await props.onImported();
      props.onClose();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const onFileChange = (e: Event) => {
    const input = e.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (file) void handleFile(file);
  };

  const onDragOver = (e: DragEvent) => {
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = "copy";
    setDragging(true);
  };
  const onDragLeave = () => setDragging(false);
  const onDrop = (e: DragEvent) => {
    e.preventDefault();
    setDragging(false);
    const file = e.dataTransfer?.files?.[0];
    if (file) void handleFile(file);
  };

  return (
    <Dialog
      open={props.open}
      onClose={close}
      title={t("env.importTitle")}
      footer={
        <Button variant="secondary" onClick={close}>
          {t("common.cancel")}
        </Button>
      }
    >
      <div class="space-y-3">
        <p class="text-sm text-ink-muted">{t("env.importHint")}</p>
        <div
          onClick={() => fileInput?.click()}
          onDragOver={onDragOver}
          onDragLeave={onDragLeave}
          onDrop={onDrop}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault();
              fileInput?.click();
            }
          }}
          role="button"
          tabIndex={0}
          class={[
            "flex cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border border-dashed px-6 py-10 text-center transition-colors",
            dragging()
              ? "border-brand bg-brand-subtle"
              : "border-line bg-surface-2 hover:border-ink-faint",
          ].join(" ")}
        >
          <Icon name="upload" size={28} class={dragging() ? "text-brand" : "text-ink-faint"} />
          <p class="text-sm font-medium text-ink">{t("env.dropTitle")}</p>
          <p class="text-xs text-ink-muted">{t("env.dropBrowse")}</p>
          <input
            ref={(el) => (fileInput = el)}
            type="file"
            accept=".env,.json,.yaml,.yml"
            onChange={onFileChange}
            class="hidden"
          />
        </div>
        <Show when={busy()}>
          <p class="text-sm text-ink-muted">{t("common.loading")}</p>
        </Show>
        <Show when={error()}>
          <p class="text-sm text-danger">{error()}</p>
        </Show>
      </div>
    </Dialog>
  );
}

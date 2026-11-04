// Export variables of an environment as JSON, .env or YAML.
import { createSignal, For, Show } from "solid-js";
import { environments } from "../../api";
import { t } from "../../lib/i18n";
import { errorMessage } from "../../lib/errors";
import { downloadBlob } from "../../lib/download";
import { Icon } from "../ui";
import { Dialog } from "../shared";

interface FormatOption {
  id: "json" | "yaml" | "env";
  ext: string;
  label: string;
}

const EXPORT_FORMATS: FormatOption[] = [
  { id: "json", ext: "json", label: "JSON" },
  { id: "env", ext: "env", label: ".env" },
  { id: "yaml", ext: "yaml", label: "YAML" },
];

export function ExportVariablesDialog(props: {
  open: boolean;
  onClose: () => void;
  envId: string;
}) {
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  const doExport = async (format: FormatOption) => {
    setBusy(true);
    setError(null);
    try {
      const { blob, filename } = await environments.exportFile(props.envId, format.id);
      downloadBlob(blob, filename);
      props.onClose();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={props.open} onClose={props.onClose} title={t("env.exportTitle")}>
      <div class="space-y-3">
        <p class="text-sm text-ink-muted">{t("env.exportHint")}</p>
        <div class="space-y-2">
          <For each={EXPORT_FORMATS}>
            {(format) => (
              <button
                type="button"
                disabled={busy()}
                onClick={() => doExport(format)}
                class="flex w-full items-center gap-3 rounded-lg border border-line bg-surface-2 px-3 py-2.5 text-left transition-colors hover:border-brand hover:bg-surface disabled:pointer-events-none disabled:opacity-60"
              >
                <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-brand-subtle text-brand">
                  <Icon name="code-bracket" size={15} />
                </div>
                <div class="min-w-0 flex-1">
                  <p class="text-sm font-medium text-ink">{format.label}</p>
                  <p class="text-xs text-ink-muted">.{format.ext}</p>
                </div>
                <Icon name="chevron-right" size={15} class="text-ink-faint" />
              </button>
            )}
          </For>
        </div>
        <Show when={error()}>
          <p class="text-sm text-danger">{error()}</p>
        </Show>
      </div>
    </Dialog>
  );
}

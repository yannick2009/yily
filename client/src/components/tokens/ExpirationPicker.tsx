// Fixed PAT lifetime presets as a radio pill group. When allowNever is true, an extra "Never" pill (value 0) is shown for read-only tokens.
import { For, Show } from "solid-js";
import { t } from "../../lib/i18n";
import { EXPIRATION_NEVER, EXPIRATION_PRESETS } from "./constants";

export function ExpirationPicker(props: {
  value: number;
  onChange: (days: number) => void;
  allowNever?: boolean;
}) {
  const pillClass = (active: boolean) =>
    [
      "rounded-md border px-3 py-1.5 text-sm font-medium transition-colors",
      active
        ? "border-brand bg-brand-subtle text-brand"
        : "border-line bg-surface text-ink-muted hover:border-ink-faint hover:text-ink",
    ].join(" ");
  return (
    <div class="flex flex-wrap gap-2">
      <For each={EXPIRATION_PRESETS}>
        {(days) => {
          const active = () => props.value === days;
          return (
            <button
              type="button"
              onClick={() => props.onChange(days)}
              class={pillClass(active())}
              aria-pressed={active()}
            >
              {t("tokens.expirationOption", { days })}
            </button>
          );
        }}
      </For>
      <Show when={props.allowNever}>
        <button
          type="button"
          onClick={() => props.onChange(EXPIRATION_NEVER)}
          class={pillClass(props.value === EXPIRATION_NEVER)}
          aria-pressed={props.value === EXPIRATION_NEVER}
        >
          {t("tokens.expirationNever")}
        </button>
      </Show>
    </div>
  );
}

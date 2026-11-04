// Two scope choices (read / write) as a radio pill group.
import { For } from "solid-js";
import { t } from "../../lib/i18n";
import type { TokenScope } from "../../types";

export function ScopePicker(props: { value: TokenScope; onChange: (scope: TokenScope) => void }) {
  const scopes: Array<{ value: TokenScope; labelKey: string; descKey: string }> = [
    { value: "read", labelKey: "tokens.scopeRead", descKey: "tokens.scopeReadDesc" },
    { value: "write", labelKey: "tokens.scopeWrite", descKey: "tokens.scopeWriteDesc" },
  ];
  return (
    <div class="grid grid-cols-2 gap-2">
      <For each={scopes}>
        {(opt) => {
          const active = () => props.value === opt.value;
          return (
            <button
              type="button"
              onClick={() => props.onChange(opt.value)}
              aria-pressed={active()}
              class={[
                "rounded-md border px-3 py-2 text-left transition-colors",
                active()
                  ? "border-brand bg-brand-subtle"
                  : "border-line bg-surface hover:border-ink-faint",
              ].join(" ")}
            >
              <p class={["text-sm font-medium", active() ? "text-brand" : "text-ink"].join(" ")}>
                {t(opt.labelKey)}
              </p>
              <p class="mt-0.5 text-xs text-ink-muted">{t(opt.descKey)}</p>
            </button>
          );
        }}
      </For>
    </div>
  );
}

// Table body + layout for the variables list. Each row shows status, value (reveal/hide/copy)
// and tags. Reveal and copy state is owned by the parent and passed through callbacks.
import { For, Show } from "solid-js";
import { A } from "@solidjs/router";
import { t } from "../../lib/i18n";
import { truncateMiddle } from "../../lib/string";
import { Icon } from "../ui";
import { RevealCountdown } from "../shared";
import type { Variable } from "../../types";
import { TagList } from "./TagList";

export function VariablesTable(props: {
  variables: Variable[];
  projectId: string;
  envId: string;
  isRevealed: (id: string) => boolean;
  getRevealedValue: (id: string) => string;
  remainingFor: (id: string) => number;
  copiedId: string | null;
  onToggleReveal: (id: string) => void;
  onCopy: (id: string) => void;
  onDelete: (target: { id: string; name: string }) => void;
}) {
  return (
    <div class="overflow-x-auto rounded-xl border border-line bg-surface">
      <table class="w-full table-fixed border-collapse text-left text-sm">
        <thead>
          <tr class="border-b border-line text-xs uppercase tracking-wide text-ink-faint">
            <th class="px-4 py-3 font-medium">{t("common.name")}</th>
            <th class="w-16 px-4 py-3 text-center font-medium">{t("env.status")}</th>
            <th class="w-64 px-4 py-3 font-medium">{t("env.value")}</th>
            <th class="px-4 py-3 font-medium">{t("project.tags")}</th>
            <th class="w-32 px-4 py-3 text-right font-medium">{t("env.actions")}</th>
          </tr>
        </thead>
        <tbody>
          <For each={props.variables}>
            {(v) => {
              const revealed = () => props.isRevealed(v.id);
              const remaining = () => props.remainingFor(v.id);
              const currentValue = () => props.getRevealedValue(v.id);
              const variableTags = () => v.tags ?? [];

              return (
                <tr class="border-t border-line first:border-t-0">
                  <td class="py-3 pl-4 pr-4 align-top">
                    <div class="flex items-center gap-2">
                      <A
                        href={`/projects/${props.projectId}/environments/${props.envId}/variables/${v.id}`}
                        class="font-mono text-sm text-ink hover:text-brand hover:underline"
                      >
                        {v.name}
                      </A>
                      <Show when={!v.active}>
                        <span class="shrink-0 rounded-md bg-warning-subtle px-1.5 py-0.5 text-xs font-medium text-warning">
                          {t("variable.needsValue")}
                        </span>
                      </Show>
                    </div>
                  </td>
                  <td class="py-3 pl-4 pr-4 text-center align-top">
                    <span
                      class={[
                        "mt-1.5 inline-block h-1.5 w-1.5 rounded-full",
                        v.active ? "bg-success" : "bg-ink-faint",
                      ].join(" ")}
                      title={v.active ? t("variable.active") : t("variable.inactive")}
                    />
                  </td>
                  <td class="py-3 pr-4 align-top">
                    <div class="flex items-center gap-2">
                      <code
                        class={[
                          "min-w-0 flex-1 truncate font-mono text-sm",
                          v.active ? "text-ink-muted" : "text-ink-faint",
                        ].join(" ")}
                        title={revealed() ? currentValue() || undefined : undefined}
                      >
                        {!v.active
                          ? "– – – – –"
                          : revealed()
                            ? truncateMiddle(currentValue() || t("env.noValue"))
                            : "••••••••••"}
                      </code>
                      <Show when={revealed() && remaining() > 0}>
                        <RevealCountdown seconds={remaining()} />
                      </Show>
                    </div>
                  </td>
                  <td class="py-3 pr-4 align-top">
                    <TagList tags={variableTags()} max={3} />
                  </td>
                  <td class="py-3 pr-4 text-right align-top">
                    <div class="flex items-center justify-end gap-1">
                      <button
                        type="button"
                        onClick={() => props.onToggleReveal(v.id)}
                        disabled={!v.active}
                        class={[
                          "rounded-md p-1.5 disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent",
                          revealed()
                            ? "bg-brand-subtle text-brand hover:text-brand-hover"
                            : "text-ink-muted hover:bg-surface-2 hover:text-ink disabled:hover:text-ink-muted",
                        ].join(" ")}
                        aria-label={
                          revealed()
                            ? t("env.hideAria", { name: v.name })
                            : t("env.revealAria", { name: v.name })
                        }
                        title={!v.active ? t("variable.needsValue") : undefined}
                      >
                        <Icon name={revealed() ? "eye-off" : "eye"} size={15} />
                      </button>
                      <button
                        type="button"
                        onClick={() => props.onCopy(v.id)}
                        disabled={!v.active}
                        class="rounded-md p-1.5 text-ink-muted hover:bg-surface-2 hover:text-ink disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-ink-muted"
                        aria-label={t("env.copyAria", { name: v.name })}
                        title={!v.active ? t("variable.needsValue") : undefined}
                      >
                        <Icon name={props.copiedId === v.id ? "check" : "copy"} size={15} />
                      </button>
                      <button
                        type="button"
                        onClick={() => props.onDelete({ id: v.id, name: v.name })}
                        class="rounded-md p-1.5 text-ink-muted hover:bg-danger-subtle hover:text-danger"
                        aria-label={t("common.deleteAria", { name: v.name })}
                      >
                        <Icon name="trash" size={15} />
                      </button>
                    </div>
                  </td>
                </tr>
              );
            }}
          </For>
        </tbody>
      </table>
    </div>
  );
}

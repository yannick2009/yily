// One column of the sort control. Clicking the active pill toggles the direction
// (indicated by the arrow); clicking the inactive one switches the sort column
// while preserving the current direction.
import { Show } from "solid-js";

export function SortPill(props: {
  label: string;
  active: boolean;
  order: "asc" | "desc";
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={props.onClick}
      aria-pressed={props.active}
      class={[
        "inline-flex items-center gap-1 rounded-md border px-3 py-1.5 text-sm font-medium transition-colors",
        props.active
          ? "border-brand bg-brand-subtle text-brand"
          : "border-line bg-surface text-ink-muted hover:border-ink-faint hover:text-ink",
      ].join(" ")}
    >
      {props.label}
      <Show when={props.active}>
        <span aria-hidden="true">{props.order === "asc" ? "↑" : "↓"}</span>
      </Show>
    </button>
  );
}

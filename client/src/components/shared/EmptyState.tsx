import type { JSX } from "solid-js";
import { Icon, type IconName } from "../ui/Icon";

// EmptyState component displays a message when there is no data to show. It includes an icon, title, description, and an optional action button.
// The component is styled with a dashed border and centered content to indicate an empty state in the UI.
export function EmptyState(props: {
  icon: IconName;
  title: string;
  description: string;
  action?: JSX.Element;
}) {
  return (
    <div class="flex flex-col items-center justify-center rounded-xl border border-dashed border-line px-6 py-16 text-center">
      <div class="mb-3 flex h-11 w-11 items-center justify-center rounded-lg bg-surface-2 text-ink-muted">
        <Icon name={props.icon} size={20} />
      </div>
      <h3 class="text-sm font-semibold text-ink">{props.title}</h3>
      <p class="mt-1 max-w-sm text-sm text-ink-muted">{props.description}</p>
      {props.action ? <div class="mt-4">{props.action}</div> : null}
    </div>
  );
}

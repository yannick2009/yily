// Displays a variable's tags on a single row, truncating to `max` items.
// Any overflow collapses into a "+N" chip whose title lists the hidden names.
import { For, Show } from "solid-js";
import { Chip } from "../ui";

export function TagList(props: { tags: { id: string; name: string }[]; max: number }) {
  const visible = () => props.tags.slice(0, props.max);
  const hidden = () => props.tags.slice(props.max);
  return (
    <div class="flex min-w-0 flex-nowrap items-center gap-1 overflow-hidden">
      <For each={visible()}>
        {(tag) => (
          <Chip variant="brand" class="shrink-0 max-w-32 truncate" title={tag.name}>
            {tag.name}
          </Chip>
        )}
      </For>
      <Show when={hidden().length > 0}>
        <Chip
          variant="neutral"
          class="shrink-0"
          title={hidden()
            .map((t) => t.name)
            .join(", ")}
        >
          +{hidden().length}
        </Chip>
      </Show>
    </div>
  );
}

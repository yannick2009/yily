// Countdown pill shown next to a revealed secret until it re-hides.
import { Icon } from "../ui";

export function RevealCountdown(props: { seconds: number }) {
  return (
    <span
      class="inline-flex shrink-0 items-center gap-1 rounded-full bg-surface-2 px-2 py-0.5 font-mono text-xs tabular-nums text-ink-muted"
      title={`${props.seconds}s`}
    >
      <Icon name="clock" size={12} />
      {props.seconds}s
    </span>
  );
}

// Compact label showing a token's scope next to its name in the list.
import { t } from "../../lib/i18n";
import type { TokenScope } from "../../types";

export function ScopeBadge(props: { scope: TokenScope }) {
  const classes = () =>
    props.scope === "write" ? "bg-brand-subtle text-brand" : "bg-surface-2 text-ink-muted";
  return (
    <span
      class={[
        "shrink-0 rounded-md px-1.5 py-0.5 text-xs font-medium uppercase tracking-wide",
        classes(),
      ].join(" ")}
    >
      {t(props.scope === "write" ? "tokens.scopeWrite" : "tokens.scopeRead")}
    </span>
  );
}

// Compact label showing a token's scope next to its name in the list.
import { Chip } from "../ui";
import { t } from "../../lib/i18n";
import type { TokenScope } from "../../types";

export function ScopeBadge(props: { scope: TokenScope }) {
  return (
    <Chip
      variant={props.scope === "write" ? "brand" : "neutral"}
      class="shrink-0 uppercase tracking-wide"
    >
      {t(props.scope === "write" ? "tokens.scopeWrite" : "tokens.scopeRead")}
    </Chip>
  );
}

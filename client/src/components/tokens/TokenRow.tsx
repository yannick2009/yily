// Row for a single token in the list: name, scope badge, expiration label, rotate/revoke actions.
import { Show } from "solid-js";
import { t } from "../../lib/i18n";
import { formatDate } from "../../lib/date";
import { Chip, Icon } from "../ui";
import type { Token } from "../../types";
import { ScopeBadge } from "./ScopeBadge";
import { EXPIRY_WARNING_DAYS, MS_PER_DAY } from "./constants";

export function TokenRow(props: {
  token: Token;
  onRotate: (token: Token) => void;
  onRevoke: (token: Token) => void;
}) {
  const revoked = () => props.token.revoked_at != null;
  const expiringSoon = () =>
    !revoked() &&
    props.token.expires_at != null &&
    daysUntil(props.token.expires_at) <= EXPIRY_WARNING_DAYS;
  const expirationLabel = () => {
    if (revoked()) return t("tokens.revoked");
    if (props.token.expires_at == null) return t("tokens.neverExpires");
    return t("tokens.expires", { date: formatDate(props.token.expires_at) });
  };

  return (
    <div class="flex items-center gap-4 px-4 py-3.5">
      <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-surface-2 text-ink-muted">
        <Icon name={revoked() ? "key-off" : "key"} size={16} />
      </div>
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <p class="truncate text-sm font-medium text-ink">{props.token.name}</p>
          <ScopeBadge scope={props.token.scope} />
          <Show when={expiringSoon()}>
            <Chip variant="danger" class="shrink-0">
              {t("tokens.expiringSoon")}
            </Chip>
          </Show>
        </div>
        <p class="truncate text-xs text-ink-muted">{expirationLabel()}</p>
        <p class="truncate text-xs text-ink-faint">
          {props.token.last_used_at
            ? t("tokens.lastUsed", { when: formatDate(props.token.last_used_at) })
            : t("tokens.neverUsed")}
        </p>
      </div>
      <div class="hidden shrink-0 text-right text-xs text-ink-muted sm:block">
        <p>{t("tokens.created", { date: formatDate(props.token.created_at) })}</p>
      </div>
      <Show
        when={!revoked()}
        fallback={
          <Chip variant="neutral" class="shrink-0">
            {t("tokens.revoked")}
          </Chip>
        }
      >
        <button
          type="button"
          class="rounded-md p-1.5 text-ink-faint hover:bg-brand-subtle hover:text-brand"
          onClick={() => props.onRotate(props.token)}
          aria-label={t("tokens.rotateAria", { name: props.token.name })}
          title={t("tokens.rotate")}
        >
          <Icon name="refresh" size={15} />
        </button>
        <button
          type="button"
          class="rounded-md p-1.5 text-ink-faint hover:bg-danger-subtle hover:text-danger"
          onClick={() => props.onRevoke(props.token)}
          aria-label={t("tokens.revokeAria", { name: props.token.name })}
          title={t("tokens.revoke")}
        >
          <Icon name="x-circle" size={15} />
        </button>
      </Show>
    </div>
  );
}

// Whole-day distance from now to the given ISO timestamp. A past timestamp yields a non-positive number.
function daysUntil(iso: string): number {
  return Math.ceil((new Date(iso).getTime() - Date.now()) / MS_PER_DAY);
}

// TokensRoute lists active PATs and orchestrates create/rotate/revoke dialogs.
// The secret value of a token is only ever shown once, right after creation or rotation.
import { createMemo, createResource, createSignal, For, Show } from "solid-js";
import { tokens } from "../api";
import { t } from "../lib/i18n";
import type { Token } from "../types";
import { Button, Card, Icon } from "../components/ui";
import { EmptyState } from "../components/shared";
import {
  CreateTokenDialog,
  RevokeTokenDialog,
  RotateTokenDialog,
  TokenRow,
} from "../components/tokens";

export function TokensRoute() {
  const [list, { refetch }] = createResource(() => tokens.list());

  const [createOpen, setCreateOpen] = createSignal(false);
  const [rotating, setRotating] = createSignal<Token | null>(null);
  const [deleting, setDeleting] = createSignal<Token | null>(null);
  const [showRevoked, setShowRevoked] = createSignal(false);

  const visibleTokens = createMemo<Token[]>(() => {
    const all = list() ?? [];
    return showRevoked() ? all : all.filter((token) => token.revoked_at == null);
  });
  const revokedCount = createMemo(
    () => (list() ?? []).filter((tk) => tk.revoked_at != null).length,
  );

  return (
    <div>
      <div class="mb-6 flex items-start justify-between gap-4">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight">{t("tokens.title")}</h1>
          <p class="mt-1 text-sm text-ink-muted">{t("tokens.subtitle")}</p>
        </div>
        <Button icon="plus" onClick={() => setCreateOpen(true)}>
          {t("tokens.create")}
        </Button>
      </div>

      <div class="mb-6 flex gap-3 rounded-xl border border-line bg-surface p-4">
        <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-brand-subtle text-brand">
          <Icon name="key" size={16} />
        </div>
        <div class="text-sm text-ink-muted">
          <p class="font-medium text-ink">{t("tokens.bannerTitle")}</p>
          <p class="mt-0.5">{t("tokens.bannerBody")}</p>
        </div>
      </div>

      <div class="mb-3 flex items-center justify-between gap-3">
        <Show when={revokedCount() > 0} fallback={<span />}>
          <button
            type="button"
            onClick={() => setShowRevoked((v) => !v)}
            class="inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
          >
            <Icon name={showRevoked() ? "eye-off" : "eye"} size={14} />
            {showRevoked()
              ? t("tokens.hideRevoked")
              : t("tokens.showRevoked", { count: revokedCount() })}
          </button>
        </Show>
        <a
          href="https://docs.yily.io"
          target="_blank"
          rel="noreferrer"
          class="inline-flex items-center gap-1.5 text-sm text-brand hover:underline"
        >
          <Icon name="book-open" size={14} />
          {t("tokens.docs")}
        </a>
      </div>

      <Show
        when={!list.loading}
        fallback={<p class="py-8 text-center text-sm text-ink-muted">{t("common.loading")}</p>}
      >
        <Show
          when={visibleTokens().length > 0}
          fallback={
            <EmptyState
              icon="key"
              title={t("tokens.emptyTitle")}
              description={t("tokens.emptyDescription")}
              action={
                <Button icon="plus" onClick={() => setCreateOpen(true)}>
                  {t("tokens.create")}
                </Button>
              }
            />
          }
        >
          <Card class="divide-y divide-line">
            <For each={visibleTokens()}>
              {(token) => <TokenRow token={token} onRotate={setRotating} onRevoke={setDeleting} />}
            </For>
          </Card>
        </Show>
      </Show>

      <CreateTokenDialog
        open={createOpen()}
        onClose={() => setCreateOpen(false)}
        onCreated={() => refetch()}
      />
      <RotateTokenDialog
        token={rotating()}
        onClose={() => setRotating(null)}
        onRotated={() => refetch()}
      />
      <RevokeTokenDialog
        token={deleting()}
        onClose={() => setDeleting(null)}
        onRevoked={() => refetch()}
      />
    </div>
  );
}

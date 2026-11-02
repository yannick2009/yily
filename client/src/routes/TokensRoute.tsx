import { createMemo, createResource, createSignal, For, Show } from "solid-js";
import { tokens } from "../api";
import { locale, t } from "../lib/i18n";
import { errorMessage } from "../lib/errors";
import type { Token } from "../types";
import { Button, Card, Icon, Input } from "../components/ui";
import { Dialog, EmptyState, Field } from "../components/shared";

// Fixed set of lifetimes offered by the API. Must match server AllowedPATTTLDays.
const EXPIRATION_PRESETS = [7, 30, 60, 90] as const;
const DEFAULT_EXPIRATION_DAYS = 30;
// Threshold (in days) below which a non-revoked token surfaces an "expiring soon" warning badge.
const EXPIRY_WARNING_DAYS = 7;
const MS_PER_DAY = 24 * 60 * 60 * 1000;

// TokensRoute displays a list of all active tokens and allows creating, rotating and revoking them.
// The secret value of a token is only ever shown once, right after creation or rotation.
export function TokensRoute() {
  const [list, { refetch }] = createResource(() => tokens.list());

  const [createOpen, setCreateOpen] = createSignal(false);
  const [name, setName] = createSignal("");
  const [createDays, setCreateDays] = createSignal<number>(DEFAULT_EXPIRATION_DAYS);
  const [created, setCreated] = createSignal<string | null>(null);
  const [copied, setCopied] = createSignal(false);
  const [deleting, setDeleting] = createSignal<Token | null>(null);
  const [rotating, setRotating] = createSignal<Token | null>(null);
  const [rotateDays, setRotateDays] = createSignal<number>(DEFAULT_EXPIRATION_DAYS);
  const [rotated, setRotated] = createSignal<string | null>(null);
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [showRevoked, setShowRevoked] = createSignal(false);

  const visibleTokens = createMemo<Token[]>(() => {
    const all = list() ?? [];
    return showRevoked() ? all : all.filter((token) => token.revoked_at == null);
  });
  const revokedCount = createMemo(() => (list() ?? []).filter((t) => t.revoked_at != null).length);

  const openCreate = () => {
    setName("");
    setCreateDays(DEFAULT_EXPIRATION_DAYS);
    setCreated(null);
    setCopied(false);
    setError(null);
    setCreateOpen(true);
  };

  const closeCreate = () => {
    setCreateOpen(false);
    setCreated(null);
    setCopied(false);
    setError(null);
  };

  const openRotate = (token: Token) => {
    setRotating(token);
    setRotateDays(DEFAULT_EXPIRATION_DAYS);
    setRotated(null);
    setCopied(false);
    setError(null);
  };

  const closeRotate = () => {
    setRotating(null);
    setRotated(null);
    setCopied(false);
    setError(null);
  };

  const submitCreate = async (e: Event) => {
    e.preventDefault();
    const n = name().trim();
    if (!n) return;
    setBusy(true);
    setError(null);
    try {
      const createdToken = await tokens.create({ name: n, expires_in_days: createDays() });
      setCreated(createdToken.token);
      setName("");
      await refetch();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const submitRotate = async (e: Event) => {
    e.preventDefault();
    const target = rotating();
    if (!target) return;
    setBusy(true);
    setError(null);
    try {
      const newToken = await tokens.rotate(target.id, { expires_in_days: rotateDays() });
      setRotated(newToken.token);
      await refetch();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.save"));
    } finally {
      setBusy(false);
    }
  };

  const copyCreated = async () => {
    const s = created();
    if (!s) return;
    try {
      await navigator.clipboard.writeText(s);
      setCopied(true);
    } catch {
      /* clipboard unavailable */
    }
  };

  const copyRotated = async () => {
    const s = rotated();
    if (!s) return;
    try {
      await navigator.clipboard.writeText(s);
      setCopied(true);
    } catch {
      /* clipboard unavailable */
    }
  };

  const confirmRevoke = async () => {
    const d = deleting();
    if (!d) return;
    setBusy(true);
    setError(null);
    try {
      await tokens.revoke(d.id);
      setDeleting(null);
      await refetch();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div>
      <div class="mb-6 flex items-start justify-between gap-4">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight">{t("tokens.title")}</h1>
          <p class="mt-1 text-sm text-ink-muted">{t("tokens.subtitle")}</p>
        </div>
        <Button icon="plus" onClick={openCreate}>
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
                <Button icon="plus" onClick={openCreate}>
                  {t("tokens.create")}
                </Button>
              }
            />
          }
        >
          <Card class="divide-y divide-line">
            <For each={visibleTokens()}>
              {(token) => {
                const revoked = () => token.revoked_at != null;
                const expiringSoon = () =>
                  !revoked() && daysUntil(token.expires_at) <= EXPIRY_WARNING_DAYS;
                return (
                  <div class="flex items-center gap-4 px-4 py-3.5">
                    <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-surface-2 text-ink-muted">
                      <Icon name="key" size={16} />
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-2">
                        <p class="truncate text-sm font-medium text-ink">{token.name}</p>
                        <Show when={expiringSoon()}>
                          <span class="shrink-0 rounded-md bg-danger-subtle px-1.5 py-0.5 text-xs font-medium text-danger">
                            {t("tokens.expiringSoon")}
                          </span>
                        </Show>
                      </div>
                      <p class="truncate text-xs text-ink-muted">
                        {revoked()
                          ? t("tokens.revoked")
                          : t("tokens.expires", { date: formatDate(token.expires_at) })}
                      </p>
                      <p class="truncate text-xs text-ink-faint">
                        {token.last_used_at
                          ? t("tokens.lastUsed", { when: formatDate(token.last_used_at) })
                          : t("tokens.neverUsed")}
                      </p>
                    </div>
                    <div class="hidden shrink-0 text-right text-xs text-ink-muted sm:block">
                      <p>{t("tokens.created", { date: formatDate(token.created_at) })}</p>
                    </div>
                    <Show
                      when={!revoked()}
                      fallback={
                        <span class="shrink-0 rounded-md bg-surface-2 px-1.5 py-0.5 text-xs font-medium text-ink-muted">
                          {t("tokens.revoked")}
                        </span>
                      }
                    >
                      <button
                        type="button"
                        class="rounded-md p-1.5 text-ink-faint hover:bg-brand-subtle hover:text-brand"
                        onClick={() => openRotate(token)}
                        aria-label={t("tokens.rotateAria", { name: token.name })}
                        title={t("tokens.rotate")}
                      >
                        <Icon name="refresh" size={15} />
                      </button>
                      <button
                        type="button"
                        class="rounded-md p-1.5 text-ink-faint hover:bg-danger-subtle hover:text-danger"
                        onClick={() => setDeleting(token)}
                        aria-label={t("tokens.revokeAria", { name: token.name })}
                        title={t("tokens.revoke")}
                      >
                        <Icon name="x-circle" size={15} />
                      </button>
                    </Show>
                  </div>
                );
              }}
            </For>
          </Card>
        </Show>
      </Show>

      {/* Dialog for creating a new token */}
      <Dialog
        open={createOpen()}
        onClose={closeCreate}
        title={created() === null ? t("tokens.createTitle") : t("tokens.createdTitle")}
        footer={
          created() === null ? (
            <>
              <Button variant="secondary" onClick={closeCreate}>
                {t("common.cancel")}
              </Button>
              <Button type="submit" form="create-token-form" loading={busy()}>
                {t("tokens.create")}
              </Button>
            </>
          ) : (
            <Button variant="secondary" onClick={closeCreate}>
              {t("tokens.done")}
            </Button>
          )
        }
      >
        <Show
          when={created() === null}
          fallback={
            <div class="space-y-4">
              <div class="rounded-lg border border-line bg-surface-2 p-3">
                <p class="break-all font-mono text-sm text-ink">{created()}</p>
              </div>
              <div class="flex items-start gap-2 rounded-lg bg-brand-subtle p-3 text-sm text-brand">
                <Icon name="lock" size={16} class="mt-0.5 shrink-0" />
                <p>{t("tokens.copyNow")}</p>
              </div>
              <Button
                variant="secondary"
                icon={copied() ? "check" : "copy"}
                onClick={copyCreated}
                class="w-full"
              >
                {copied() ? t("tokens.copied") : t("tokens.copy")}
              </Button>
            </div>
          }
        >
          <form id="create-token-form" onSubmit={submitCreate} class="space-y-4">
            <Field label={t("common.name")} hint={t("tokens.nameHint")}>
              <Input
                value={name()}
                onInput={(e) => setName(e.currentTarget.value)}
                placeholder={t("tokens.namePlaceholder")}
              />
            </Field>
            <Field label={t("tokens.expirationLabel")} hint={t("tokens.expirationHint")}>
              <ExpirationPicker value={createDays()} onChange={setCreateDays} />
            </Field>
            <p class="text-sm text-ink-muted">{t("tokens.accessHint")}</p>
            <Show when={error()}>
              <p class="text-sm text-danger">{error()}</p>
            </Show>
          </form>
        </Show>
      </Dialog>

      {/* Dialog for rotating a token */}
      <Dialog
        open={!!rotating()}
        onClose={closeRotate}
        title={rotated() === null ? t("tokens.rotateTitle") : t("tokens.rotatedTitle")}
        footer={
          rotated() === null ? (
            <>
              <Button variant="secondary" onClick={closeRotate}>
                {t("common.cancel")}
              </Button>
              <Button type="submit" form="rotate-token-form" loading={busy()}>
                {t("tokens.rotate")}
              </Button>
            </>
          ) : (
            <Button variant="secondary" onClick={closeRotate}>
              {t("tokens.done")}
            </Button>
          )
        }
      >
        <Show
          when={rotated() === null}
          fallback={
            <div class="space-y-4">
              <div class="rounded-lg border border-line bg-surface-2 p-3">
                <p class="break-all font-mono text-sm text-ink">{rotated()}</p>
              </div>
              <div class="flex items-start gap-2 rounded-lg bg-brand-subtle p-3 text-sm text-brand">
                <Icon name="lock" size={16} class="mt-0.5 shrink-0" />
                <p>{t("tokens.copyNow")}</p>
              </div>
              <Button
                variant="secondary"
                icon={copied() ? "check" : "copy"}
                onClick={copyRotated}
                class="w-full"
              >
                {copied() ? t("tokens.copied") : t("tokens.copy")}
              </Button>
            </div>
          }
        >
          <form id="rotate-token-form" onSubmit={submitRotate} class="space-y-4">
            <p class="text-sm text-ink-muted">
              {t("tokens.rotateConfirm", { name: rotating()?.name ?? "" })}
            </p>
            <Field label={t("tokens.expirationLabel")} hint={t("tokens.expirationHint")}>
              <ExpirationPicker value={rotateDays()} onChange={setRotateDays} />
            </Field>
            <Show when={error()}>
              <p class="text-sm text-danger">{error()}</p>
            </Show>
          </form>
        </Show>
      </Dialog>

      {/* Dialog for confirming token revocation */}
      <Dialog
        open={!!deleting()}
        onClose={() => setDeleting(null)}
        title={t("tokens.revokeTitle")}
        footer={
          <>
            <Button variant="secondary" onClick={() => setDeleting(null)}>
              {t("common.cancel")}
            </Button>
            <Button variant="danger" onClick={confirmRevoke} loading={busy()}>
              {t("tokens.revoke")}
            </Button>
          </>
        }
      >
        <p class="text-sm text-ink-muted">
          {t("tokens.revokeConfirm", { name: deleting()?.name ?? "" })}
        </p>
        <Show when={error()}>
          <p class="mt-3 text-sm text-danger">{error()}</p>
        </Show>
      </Dialog>
    </div>
  );
}

// ExpirationPicker renders the fixed PAT lifetime presets as a radio pill group.
function ExpirationPicker(props: { value: number; onChange: (days: number) => void }) {
  return (
    <div class="flex flex-wrap gap-2">
      <For each={EXPIRATION_PRESETS}>
        {(days) => {
          const active = () => props.value === days;
          return (
            <button
              type="button"
              onClick={() => props.onChange(days)}
              class={[
                "rounded-md border px-3 py-1.5 text-sm font-medium transition-colors",
                active()
                  ? "border-brand bg-brand-subtle text-brand"
                  : "border-line bg-surface text-ink-muted hover:border-ink-faint hover:text-ink",
              ].join(" ")}
              aria-pressed={active()}
            >
              {t("tokens.expirationOption", { days })}
            </button>
          );
        }}
      </For>
    </div>
  );
}

// Formats an ISO date using the current locale.
function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(locale(), {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

// daysUntil returns the whole-day distance from now to the given ISO timestamp. A past timestamp yields a non-positive number.
function daysUntil(iso: string): number {
  return Math.ceil((new Date(iso).getTime() - Date.now()) / MS_PER_DAY);
}

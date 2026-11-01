import { createResource, createSignal, For, Show } from "solid-js";
import { tokens } from "../api";
import { locale, t } from "../lib/i18n";
import type { Token } from "../types";
import { Button, Card, Icon, Input } from "../components/ui";
import { Dialog, EmptyState, Field } from "../components/shared";

// TokensRoute displays a list of all active tokens and allows creating new ones.
// The secret value of a token is only ever shown once, at creation.
export function TokensRoute() {
  const [list, { refetch }] = createResource(() => tokens.list());

  const [createOpen, setCreateOpen] = createSignal(false);
  const [name, setName] = createSignal("");
  const [created, setCreated] = createSignal<string | null>(null);
  const [copied, setCopied] = createSignal(false);
  const [deleting, setDeleting] = createSignal<Token | null>(null);
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  const openCreate = () => {
    setName("");
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

  const submitCreate = async (e: Event) => {
    e.preventDefault();
    const n = name().trim();
    if (!n) return;
    setBusy(true);
    setError(null);
    try {
      const createdToken = await tokens.create({ name: n });
      setCreated(createdToken.token);
      setName("");
      await refetch();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.save"));
    } finally {
      setBusy(false);
    }
  };

  const copySecret = async () => {
    const s = created();
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
      setError(err instanceof Error ? err.message : t("common.delete"));
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

      <div class="mb-3 flex justify-end">
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
          when={(list() ?? []).length > 0}
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
            <For each={list() ?? []}>
              {(token) => {
                const revoked = () => token.revoked_at != null;
                return (
                  <div class="flex items-center gap-4 px-4 py-3.5">
                    <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-surface-2 text-ink-muted">
                      <Icon name="key" size={16} />
                    </div>
                    <div class="min-w-0 flex-1">
                      <p class="truncate text-sm font-medium text-ink">{token.name}</p>
                      <p class="truncate text-xs text-ink-muted">
                        {revoked()
                          ? t("tokens.revoked")
                          : t("tokens.expires", { date: formatDate(token.expires_at) })}
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
                        class="rounded-md p-1.5 text-ink-faint hover:bg-danger-subtle hover:text-danger"
                        onClick={() => setDeleting(token)}
                        aria-label={t("tokens.revokeAria", { name: token.name })}
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
                onClick={copySecret}
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
            <p class="text-sm text-ink-muted">{t("tokens.accessHint")}</p>
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

// Formats an ISO date using the current locale.
function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(locale(), {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

// Access tokens (mock only): create (secret shown once) and revoke.
import { createSignal, For, Show } from "solid-js";
import { tokenSecrets } from "../store/mock";
import { addToken, db, removeToken } from "../store";
import { locale, t } from "../lib/i18n";
import type { Token } from "../types";
import { Button, Card, Icon, Input } from "../components/ui";
import { Dialog, EmptyState, Field } from "../components/shared";

// TokensRoute component manages the display and interaction with API tokens. It allows users to create new tokens, view existing tokens, copy token secrets, and revoke tokens. The component uses dialogs for creating and revoking tokens, and it provides feedback to the user through messages and visual cues.
// It also handles localization of text and formatting of dates based on the current language setting.
export function TokensRoute() {
  const [createOpen, setCreateOpen] = createSignal(false);
  const [name, setName] = createSignal("");
  const [created, setCreated] = createSignal<string | null>(null);
  const [copied, setCopied] = createSignal(false);
  const [deleting, setDeleting] = createSignal<Token | null>(null);

  const openCreate = () => {
    setName("");
    setCreated(null);
    setCopied(false);
    setCreateOpen(true);
  };

  const closeCreate = () => {
    setCreateOpen(false);
    setCreated(null);
    setCopied(false);
  };

  const submitCreate = (e: Event) => {
    e.preventDefault();
    const n = name().trim();
    if (!n) return;
    const id = addToken(n);
    setCreated(tokenSecrets[id] ?? "");
    setName("");
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

  return (
    <div>
      <div class="mb-6 flex items-start justify-between gap-4">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight">
            {t("tokens.title")}
          </h1>
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
          class="inline-flex items-center gap-1.5 text-sm text-brand hover:underline">
          <Icon name="book-open" size={14} />
          {t("tokens.docs")}
        </a>
      </div>

      <Show
        when={db.tokens.length > 0}
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
        }>
        <Card class="divide-y divide-line">
          <For each={db.tokens}>
            {(token) => {
              return (
                <div class="flex items-center gap-4 px-4 py-3.5">
                  <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-surface-2 text-ink-muted">
                    <Icon name="key" size={16} />
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="truncate text-sm font-medium text-ink">
                      {token.name}
                    </p>
                    <p class="truncate text-xs text-ink-muted">
                      {token.lastUsedAt
                        ? t("tokens.lastUsed", {
                            when: timeAgo(token.lastUsedAt),
                          })
                        : t("tokens.neverUsed")}
                    </p>
                  </div>
                  <div class="hidden shrink-0 text-right text-xs text-ink-muted sm:block">
                    <p>
                      {t("tokens.created", {
                        date: formatDate(token.createdAt),
                      })}
                    </p>
                  </div>
                  <button
                    type="button"
                    class="rounded-md p-1.5 text-ink-faint hover:bg-danger-subtle hover:text-danger"
                    onClick={() => setDeleting(token)}
                    aria-label={t("tokens.revokeAria", { name: token.name })}>
                    <Icon name="x-circle" size={15} />
                  </button>
                </div>
              );
            }}
          </For>
        </Card>
      </Show>

      {/* Dialog for creating a new token */}
      <Dialog
        open={createOpen()}
        onClose={closeCreate}
        title={
          created() === null
            ? t("tokens.createTitle")
            : t("tokens.createdTitle")
        }
        footer={
          created() === null ? (
            <>
              <Button variant="secondary" onClick={closeCreate}>
                {t("common.cancel")}
              </Button>
              <Button type="submit" form="create-token-form">
                {t("tokens.create")}
              </Button>
            </>
          ) : (
            <Button variant="secondary" onClick={closeCreate}>
              {t("tokens.done")}
            </Button>
          )
        }>
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
                class="w-full">
                {copied() ? t("tokens.copied") : t("tokens.copy")}
              </Button>
            </div>
          }>
          <form
            id="create-token-form"
            onSubmit={submitCreate}
            class="space-y-4">
            <Field label={t("common.name")} hint={t("tokens.nameHint")}>
              <Input
                value={name()}
                onInput={(e) => setName(e.currentTarget.value)}
                placeholder={t("tokens.namePlaceholder")}
              />
            </Field>
            <p class="text-sm text-ink-muted">{t("tokens.accessHint")}</p>
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
            <Button
              variant="danger"
              onClick={() => {
                const d = deleting();
                if (d) {
                  removeToken(d.id);
                  setDeleting(null);
                }
              }}>
              {t("tokens.revoke")}
            </Button>
          </>
        }>
        <p class="text-sm text-ink-muted">
          {t("tokens.revokeConfirm", { name: deleting()?.name ?? "" })}
        </p>
      </Dialog>
    </div>
  );
}

// Formats an ISO date string into a human-readable date format based on the current locale. It uses the `toLocaleDateString` method to format the date with short month, numeric day, and numeric year.
function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(locale(), {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

// Returns a human-readable string representing the time elapsed since the given ISO date string. It calculates the difference in minutes, hours, and days, and returns a localized string indicating how long ago the event occurred. If the event was more than 30 days ago, it falls back to formatting the date.
function timeAgo(iso: string): string {
  const ms = Date.now() - new Date(iso).getTime();
  const m = Math.floor(ms / 60000);
  if (m < 1) return t("time.justNow");
  if (m < 60) return t("time.minutesAgo", { n: m });
  const h = Math.floor(m / 60);
  if (h < 24) return t("time.hoursAgo", { n: h });
  const d = Math.floor(h / 24);
  if (d < 30) return t("time.daysAgo", { n: d });
  return formatDate(iso);
}

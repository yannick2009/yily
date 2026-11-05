// Audit log: a read-only history of actions across projects, variables and tokens.
import { createEffect, createResource, createSignal, For, on, Show, onCleanup } from "solid-js";
import { useSearchParams } from "@solidjs/router";
import { audit } from "../api";
import { currentUser } from "../store";
import { t } from "../lib/i18n";
import { formatDate, timeAgo } from "../lib/date";
import type { AuditEvent } from "../types";
import { Badge, Button, Icon, Input } from "../components/ui";
import { Dialog, EmptyState } from "../components/shared";
import { downloadBlob } from "../lib/download";
import { errorMessage } from "../lib/errors";

const ACTIONS = [
  "created",
  "updated",
  "deleted",
  "revealed",
  "export",
  "revoked",
  "rotated",
  "signed_in",
  "signed_out",
];

// Page size for the audit log listing. It matches the server's default page
// size so a "page" means the same thing on both sides.
const PAGE_SIZE = 50;

export function AuditRoute() {
  const [query, setQuery] = createSignal("");
  const [action, setAction] = createSignal<string>("all");
  const [selected, setSelected] = createSignal<AuditEvent | null>(null);
  const [from, setFrom] = createSignal("");
  const [to, setTo] = createSignal("");

  // Text search (resource/actor name), debounced and applied server-side.
  const [debouncedQuery, setDebouncedQuery] = createSignal("");
  let searchTimer: ReturnType<typeof setTimeout> | undefined;

  // Handle input events on the search box, debouncing the server request.
  const onSearchInput = (e: Event) => {
    const value = (e.currentTarget as HTMLInputElement).value;
    setQuery(value);
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => setDebouncedQuery(value), 250);
  };

  onCleanup(() => clearTimeout(searchTimer));

  // Pagination state lives in the URL (?page=N) so reloading or opening the
  // link again keeps the current page instead of falling back to the first one.
  const [searchParams, setSearchParams] = useSearchParams<{ page?: string }>();
  const page = () => {
    const n = Number.parseInt(String(searchParams.page ?? ""), 10);
    return Number.isFinite(n) && n >= 1 ? n : 1;
  };
  const setPage = (next: number | ((p: number) => number)) => {
    const value = typeof next === "function" ? next(page()) : next;
    // Page 1 is the default, so drop the parameter instead of storing ?page=1.
    setSearchParams({ page: value > 1 ? String(value) : undefined }, { replace: true });
  };

  // Any change to a filter returns the list to the first page before the next
  // fetch, so a stale page number never produces an empty view. `defer` skips
  // the initial run, preserving a page restored from the URL on reload.
  createEffect(on([action, debouncedQuery, from, to], () => setPage(1), { defer: true }));

  // Fetch the filtered audit events from the server. The query is debounced to avoid
  // sending a request on every keystroke. The action filter is applied immediately.
  const [eventsRes] = createResource(
    () => ({
      action: action(),
      search: debouncedQuery(),
      from: from(),
      to: to(),
      page: page(),
    }),
    ({ action: actionFilter, search, from: fromDate, to: toDate, page: p }) =>
      audit.list({
        action: actionFilter === "all" ? undefined : actionFilter,
        search: search.trim() || undefined,
        from: fromDate || undefined,
        to: toDate || undefined,
        limit: PAGE_SIZE,
        offset: (p - 1) * PAGE_SIZE,
      }),
  );
  const events = () => eventsRes()?.events ?? [];
  const total = () => eventsRes()?.total ?? 0;
  const totalPages = () => Math.max(1, Math.ceil(total() / PAGE_SIZE));
  const showingFrom = () => (total() === 0 ? 0 : (page() - 1) * PAGE_SIZE + 1);
  const showingTo = () => Math.min(page() * PAGE_SIZE, total());

  // Export: download the currently filtered events as a CSV file.
  const [exportBusy, setExportBusy] = createSignal(false);
  const [exportError, setExportError] = createSignal("");

  const doExport = async () => {
    setExportBusy(true);
    setExportError("");
    try {
      const { blob, filename } = await audit.exportCSV({
        action: action() === "all" ? undefined : action(),
        search: debouncedQuery().trim() || undefined,
        from: from() || undefined,
        to: to() || undefined,
      });
      downloadBlob(blob, filename);
    } catch (err) {
      setExportError(errorMessage(err));
    } finally {
      setExportBusy(false);
    }
  };

  return (
    <div>
      <div class="mb-6 flex items-start justify-between gap-4">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight">{t("audit.title")}</h1>
          <p class="mt-1 text-sm text-ink-muted">{t("audit.subtitle")}</p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          icon="download"
          loading={exportBusy()}
          onClick={doExport}
        >
          {t("audit.export")}
          <span class="rounded border border-line bg-surface-2 px-1 font-mono text-[10px] font-semibold leading-4 text-ink-muted">
            CSV
          </span>
        </Button>
      </div>

      <Show when={exportError()}>
        <p class="mb-4 text-sm text-danger">{exportError()}</p>
      </Show>

      <div class="mb-4 space-y-3">
        <div class="flex flex-wrap items-end gap-3">
          <div class="relative w-full sm:w-72">
            <Input
              value={query()}
              onInput={onSearchInput}
              placeholder={t("audit.searchPlaceholder")}
              style={{ "padding-left": "2.5rem" }}
            />
            <span class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-ink-faint">
              <Icon name="search" size={16} />
            </span>
          </div>
          <div class="flex items-end gap-2">
            <label class="flex flex-col gap-1 text-xs text-ink-muted">
              <span class="font-medium uppercase tracking-wide text-ink-faint">
                {t("audit.from")}
              </span>
              <Input
                type="date"
                value={from()}
                max={to() || undefined}
                onInput={(e) => setFrom(e.currentTarget.value)}
              />
            </label>
            <label class="flex flex-col gap-1 text-xs text-ink-muted">
              <span class="font-medium uppercase tracking-wide text-ink-faint">
                {t("audit.to")}
              </span>
              <Input
                type="date"
                value={to()}
                min={from() || undefined}
                onInput={(e) => setTo(e.currentTarget.value)}
              />
            </label>
            <Show when={from() || to()}>
              <button
                type="button"
                onClick={() => {
                  setFrom("");
                  setTo("");
                }}
                class="h-9 shrink-0 rounded-md px-2 text-sm text-ink-muted hover:bg-surface-2 hover:text-ink"
              >
                {t("audit.resetDates")}
              </button>
            </Show>
          </div>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <span class="mr-1 text-xs font-medium uppercase tracking-wide text-ink-faint">
            {t("env.filter")}
          </span>
          <button
            type="button"
            onClick={() => setAction("all")}
            class={chipClass(action() === "all")}
          >
            {t("audit.all")}
          </button>
          <For each={ACTIONS}>
            {(a) => (
              <button type="button" onClick={() => setAction(a)} class={chipClass(action() === a)}>
                {t(`audit.action.${a}`)}
              </button>
            )}
          </For>
        </div>
      </div>

      <Show
        when={!eventsRes.loading}
        fallback={<p class="py-8 text-center text-sm text-ink-muted">{t("common.loading")}</p>}
      >
        <Show
          when={events().length > 0}
          fallback={
            <EmptyState
              icon="clock"
              title={t("audit.emptyTitle")}
              description={t("audit.emptyDescription")}
            />
          }
        >
          <div class="overflow-x-auto rounded-xl border border-line bg-surface">
            <table class="w-full min-w-140 border-collapse text-left text-sm">
              <thead>
                <tr class="border-b border-line text-xs uppercase tracking-wide text-ink-faint">
                  <th class="px-4 py-3 font-medium">{t("audit.actor")}</th>
                  <th class="px-4 py-3 font-medium">{t("audit.action")}</th>
                  <th class="px-4 py-3 font-medium">{t("audit.resource")}</th>
                  <th class="px-4 py-3 text-right font-medium">{t("audit.time")}</th>
                  <th class="w-10 px-2 py-3" aria-hidden="true" />
                </tr>
              </thead>
              <tbody>
                <For each={events()}>
                  {(e) => (
                    <tr class="border-t border-line first:border-t-0">
                      <td class="py-3 pl-4 pr-4 align-top">
                        <span class="font-medium text-ink">{actorName(e)}</span>
                      </td>
                      <td class="py-3 pl-4 pr-4 align-top">
                        <Badge variant={actionVariant(e.action)}>
                          {t(`audit.action.${e.action}`)}
                        </Badge>
                      </td>
                      <td class="py-3 pl-4 pr-4 align-top">
                        <Show
                          when={e.resource_type}
                          fallback={<span class="text-ink-faint">—</span>}
                        >
                          <p class="truncate font-medium text-ink">{e.resource_name}</p>
                          <p class="mt-0.5 text-xs text-ink-muted">
                            {t(`audit.resource.${e.resource_type}`)}
                          </p>
                        </Show>
                      </td>
                      <td class="py-3 pl-4 pr-2 text-right align-top text-xs text-ink-faint">
                        <span title={formatDate(e.created_at)}>{timeAgo(e.created_at)}</span>
                      </td>
                      <td class="py-3 pl-2 pr-4 text-right align-top">
                        <button
                          type="button"
                          onClick={() => setSelected(e)}
                          class="rounded-md p-1 text-ink-faint hover:bg-surface-2 hover:text-ink"
                          aria-label={t("audit.viewDetails")}
                          title={t("audit.viewDetails")}
                        >
                          <Icon name="info" size={15} />
                        </button>
                      </td>
                    </tr>
                  )}
                </For>
              </tbody>
            </table>
          </div>

          <div class="mt-3 flex items-center justify-between gap-4">
            <p class="text-sm text-ink-muted">
              {t("audit.showing", {
                from: showingFrom(),
                to: showingTo(),
                total: total(),
              })}
            </p>
            <Show when={totalPages() > 1}>
              <div class="flex items-center gap-2">
                <Button
                  size="sm"
                  variant="ghost"
                  icon="chevron-left"
                  disabled={page() <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  {t("audit.prevPage")}
                </Button>
                <span class="text-sm text-ink-muted">
                  {t("audit.pageOf", { page: page(), total: totalPages() })}
                </span>
                <Button
                  size="sm"
                  variant="ghost"
                  iconRight="chevron-right"
                  disabled={page() >= totalPages()}
                  onClick={() => setPage((p) => Math.min(totalPages(), p + 1))}
                >
                  {t("audit.nextPage")}
                </Button>
              </div>
            </Show>
          </div>
        </Show>
      </Show>

      <Show keyed when={selected()}>
        {(e) => (
          <Dialog open onClose={() => setSelected(null)} title={t("audit.details")}>
            <dl class="space-y-2.5 text-sm">
              <div class="flex items-start justify-between gap-4">
                <dt class="shrink-0 text-xs font-medium uppercase tracking-wide text-ink-faint">
                  {t("audit.actor")}
                </dt>
                <dd class="min-w-0 text-right text-ink">{actorName(e)}</dd>
              </div>
              <div class="flex items-start justify-between gap-4">
                <dt class="shrink-0 text-xs font-medium uppercase tracking-wide text-ink-faint">
                  {t("audit.action")}
                </dt>
                <dd class="min-w-0 text-right">
                  <Badge variant={actionVariant(e.action)}>{t(`audit.action.${e.action}`)}</Badge>
                </dd>
              </div>
              <div class="flex items-start justify-between gap-4">
                <dt class="shrink-0 text-xs font-medium uppercase tracking-wide text-ink-faint">
                  {t("audit.resource")}
                </dt>
                <dd class="min-w-0 truncate text-right text-ink">
                  <Show when={e.resource_type} fallback={<span class="text-ink-faint">—</span>}>
                    {e.resource_name || "—"}
                  </Show>
                </dd>
              </div>
              <Show when={e.resource_type}>
                <div class="flex items-start justify-between gap-4">
                  <dt class="shrink-0 text-xs font-medium uppercase tracking-wide text-ink-faint">
                    {t("audit.resourceType")}
                  </dt>
                  <dd class="min-w-0 text-right text-ink">
                    {t(`audit.resource.${e.resource_type}`)}
                  </dd>
                </div>
              </Show>
              <div class="flex items-start justify-between gap-4">
                <dt class="shrink-0 text-xs font-medium uppercase tracking-wide text-ink-faint">
                  {t("audit.time")}
                </dt>
                <dd class="min-w-0 text-right text-ink">{formatDate(e.created_at)}</dd>
              </div>
            </dl>

            <h3 class="mb-2 mt-5 text-xs font-medium uppercase tracking-wide text-ink-faint">
              {t("audit.metadata")}
            </h3>
            <Show
              when={metaEntries(e).length > 0}
              fallback={<p class="text-sm text-ink-faint">{t("audit.metadataEmpty")}</p>}
            >
              <dl class="divide-y divide-line overflow-hidden rounded-lg border border-line">
                <For each={metaEntries(e)}>
                  {([key, value]) => (
                    <div class="flex items-start justify-between gap-4 px-3 py-2">
                      <dt class="font-mono text-xs text-ink-muted">{key}</dt>
                      <dd class="min-w-0 break-all text-right text-sm text-ink">{value}</dd>
                    </div>
                  )}
                </For>
              </dl>
            </Show>
          </Dialog>
        )}
      </Show>
    </div>
  );
}

// Resolve a friendly actor name from the event. The server snapshots the actor's
// name at record time (`actor_name`), so we only fall back to a generic label
// for events written before that field existed.
function actorName(e: AuditEvent): string {
  if (e.actor_type === "system") return t("audit.actor.system");
  if (e.actor_name) return e.actor_name;
  return e.actor_type === "token" ? t("audit.actor.token") : currentUser()?.name || "Admin";
}

// Color-code the action badge for quick scanning.
function actionVariant(action: string): "neutral" | "brand" | "success" {
  if (action === "created") return "success";
  if (action === "revealed" || action === "export" || action === "rotated") return "brand";
  return "neutral";
}

function chipClass(active: boolean): string {
  return [
    "rounded-full px-3 py-1 text-xs font-medium transition-colors",
    active ? "bg-brand text-on-brand" : "bg-surface-2 text-ink-muted hover:text-ink",
  ].join(" ");
}

// Parse the metadata JSON into sorted key/value pairs for the detail dialog.
function metaEntries(e: AuditEvent): [string, string][] {
  try {
    const parsed = JSON.parse(e.metadata) as Record<string, unknown>;
    return Object.entries(parsed)
      .map(([k, v]) => [k, typeof v === "string" ? v : JSON.stringify(v)] as [string, string])
      .sort((a, b) => a[0].localeCompare(b[0]));
  } catch {
    return e.metadata.trim() ? [["raw", e.metadata]] : [];
  }
}

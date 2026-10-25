// Audit log: a read-only history of actions across projects, variables and tokens.
import { createMemo, createSignal, For, Show } from "solid-js";
import { currentUser, db } from "../store";
import { locale, t } from "../lib/i18n";
import type { AuditEvent } from "../types";
import { Badge, Icon, Input } from "../components/ui";
import { Dialog, EmptyState } from "../components/shared";

const ACTIONS = ["created", "updated", "deleted", "revealed", "copied", "exported", "signed_in"];

export function AuditRoute() {
  const [query, setQuery] = createSignal("");
  const [action, setAction] = createSignal<string>("all");
  const [selected, setSelected] = createSignal<AuditEvent | null>(null);

  const events = createMemo(() =>
    [...db.auditEvents].sort((a, b) => b.createdAt.localeCompare(a.createdAt)),
  );

  const filtered = createMemo(() => {
    const q = query().trim().toLowerCase();
    return events().filter((e) => {
      if (action() !== "all" && e.action !== action()) return false;
      if (!q) return true;
      const haystack = [
        actorName(e),
        e.resourceName,
        t(`audit.resource.${e.resourceType}`),
        t(`audit.action.${e.action}`),
      ]
        .join(" ")
        .toLowerCase();
      return haystack.includes(q);
    });
  });

  return (
    <div>
      <div class="mb-6">
        <h1 class="text-2xl font-semibold tracking-tight">{t("audit.title")}</h1>
        <p class="mt-1 text-sm text-ink-muted">{t("audit.subtitle")}</p>
      </div>

      <div class="mb-4 space-y-3">
        <div class="relative w-full sm:w-72">
          <Input
            value={query()}
            onInput={(e) => setQuery(e.currentTarget.value)}
            placeholder={t("audit.searchPlaceholder")}
            style={{ "padding-left": "2.5rem" }}
          />
          <span class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-ink-faint">
            <Icon name="search" size={16} />
          </span>
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
        when={filtered().length > 0}
        fallback={
          <EmptyState
            icon="clock"
            title={t("audit.emptyTitle")}
            description={t("audit.emptyDescription")}
          />
        }
      >
        <div class="overflow-x-auto rounded-xl border border-line bg-surface">
          <table class="w-full min-w-[560px] border-collapse text-left text-sm">
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
              <For each={filtered()}>
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
                      <Show when={e.resourceType} fallback={<span class="text-ink-faint">—</span>}>
                        <p class="truncate font-medium text-ink">{e.resourceName}</p>
                        <p class="mt-0.5 text-xs text-ink-muted">
                          {t(`audit.resource.${e.resourceType}`)}
                        </p>
                      </Show>
                    </td>
                    <td class="py-3 pl-4 pr-2 text-right align-top text-xs text-ink-faint">
                      <span title={formatDate(e.createdAt)}>{timeAgo(e.createdAt)}</span>
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
                <dd class="min-w-0 text-right text-ink">
                  {e.resourceType
                    ? `${e.resourceName} · ${t(`audit.resource.${e.resourceType}`)}`
                    : "—"}
                </dd>
              </div>
              <div class="flex items-start justify-between gap-4">
                <dt class="shrink-0 text-xs font-medium uppercase tracking-wide text-ink-faint">
                  {t("audit.time")}
                </dt>
                <dd class="min-w-0 text-right text-ink">{formatDate(e.createdAt)}</dd>
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

// Resolve a friendly actor name from the event's actor id.
function actorName(e: AuditEvent): string {
  if (e.actorType === "system") return t("audit.actor.system");
  if (e.actorType === "token") return db.tokens.find((t) => t.id === e.actorId)?.name ?? e.actorId;
  return currentUser().name || "Admin";
}

// Color-code the action badge for quick scanning.
function actionVariant(action: string): "neutral" | "brand" | "success" {
  if (action === "created") return "success";
  if (action === "revealed" || action === "exported") return "brand";
  return "neutral";
}

function chipClass(active: boolean): string {
  return [
    "rounded-full px-3 py-1 text-xs font-medium transition-colors",
    active ? "bg-brand text-on-brand" : "bg-surface-2 text-ink-muted hover:text-ink",
  ].join(" ");
}

// Formats the time elapsed since an ISO date; falls back to a date for older items.
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

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(locale(), {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
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

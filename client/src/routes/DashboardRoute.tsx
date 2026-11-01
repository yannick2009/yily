// Dashboard: at-a-glance overview of projects, environments, secrets and tokens,
// plus the most recent audit activity.
import { createMemo, createResource, For, Show } from "solid-js";
import { A } from "@solidjs/router";
import { dashboard } from "../api";
import { currentUser } from "../store";
import { locale, plural, t } from "../lib/i18n";
import type { AuditEvent } from "../types";
import { Badge, Card, Icon, type IconName } from "../components/ui";

interface Stat {
  label: string;
  value: number;
  href?: string;
  icon: IconName;
}

export function DashboardRoute() {
  const [data] = createResource(() => dashboard.get());

  const stats = createMemo<Stat[]>(() => [
    {
      label: t("nav.projects"),
      value: data()?.stats.projects ?? 0,
      href: "/projects",
      icon: "folder",
    },
    {
      label: t("dashboard.environments"),
      value: data()?.stats.environments ?? 0,
      icon: "globe-alt",
    },
    {
      label: t("dashboard.secrets"),
      value: data()?.stats.variables ?? 0,
      icon: "lock",
    },
    {
      label: t("nav.tokens"),
      value: data()?.stats.tokens ?? 0,
      href: "/tokens",
      icon: "key",
    },
  ]);

  const recentActivity = () => data()?.recent_activity ?? [];
  const projects = () => data()?.projects ?? [];

  return (
    <div>
      <div class="mb-6">
        <h1 class="text-2xl font-semibold tracking-tight">{t("dashboard.title")}</h1>
        <p class="mt-1 text-sm text-ink-muted">{t("dashboard.subtitle")}</p>
      </div>

      <Show
        when={!data.loading}
        fallback={<p class="py-8 text-center text-sm text-ink-muted">{t("common.loading")}</p>}
      >
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <For each={stats()}>{(stat) => <StatCard stat={stat} />}</For>
        </div>

        <div class="mt-8 grid gap-6 lg:grid-cols-3">
          <section class="lg:col-span-2">
            <h2 class="mb-3 text-sm font-semibold uppercase tracking-wide text-ink-faint">
              {t("dashboard.recentActivity")}
            </h2>
            <Card class="divide-y divide-line">
              <Show
                when={recentActivity().length > 0}
                fallback={
                  <p class="p-4 text-sm text-ink-muted">{t("dashboard.recentActivityEmpty")}</p>
                }
              >
                <For each={recentActivity()}>
                  {(e) => (
                    <div class="flex items-center gap-3 px-4 py-3">
                      <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-surface-2 text-ink-muted">
                        <Icon name={resourceIcon(e.resource_type)} size={15} />
                      </div>
                      <div class="min-w-0 flex-1">
                        <p class="truncate text-sm font-medium text-ink">
                          {e.resource_name || t(`audit.action.${e.action}`)}
                        </p>
                        <p class="truncate text-xs text-ink-muted">
                          {t(`audit.action.${e.action}`)} · {actorLabel(e)}
                        </p>
                      </div>
                      <Badge variant={actionVariant(e.action)}>
                        {t(`audit.action.${e.action}`)}
                      </Badge>
                      <span class="hidden shrink-0 text-xs text-ink-faint sm:block">
                        {timeAgo(e.created_at)}
                      </span>
                    </div>
                  )}
                </For>
              </Show>
            </Card>
          </section>

          <section>
            <div class="mb-3 flex items-center justify-between">
              <h2 class="text-sm font-semibold uppercase tracking-wide text-ink-faint">
                {t("nav.projects")}
              </h2>
              <A href="/projects" class="text-xs text-brand hover:underline">
                {t("dashboard.viewAll")}
              </A>
            </div>
            <Card class="divide-y divide-line">
              <Show
                when={projects().length > 0}
                fallback={<p class="p-4 text-sm text-ink-muted">{t("projects.emptyTitle")}</p>}
              >
                <For each={projects()}>
                  {(p) => (
                    <A
                      href={`/projects/${p.id}`}
                      class="flex items-center gap-3 px-4 py-3 transition-colors hover:bg-surface-2"
                    >
                      <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-brand-subtle text-brand">
                        <Icon name="folder" size={15} />
                      </div>
                      <div class="min-w-0 flex-1">
                        <p class="truncate text-sm font-medium text-ink">{p.name}</p>
                        <p class="truncate text-xs text-ink-muted">
                          {plural("project.variables", p.variable_count)} ·{" "}
                          {plural("projects.environments", p.environment_count)}
                        </p>
                      </div>
                      <span class="shrink-0 text-xs text-ink-faint">{timeAgo(p.updated_at)}</span>
                    </A>
                  )}
                </For>
              </Show>
            </Card>
          </section>
        </div>
      </Show>
    </div>
  );
}

// A summary tile: clickable when it has a destination, otherwise a plain stat.
function StatCard(props: { stat: Stat }) {
  const icon = (
    <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-subtle text-brand">
      <Icon name={props.stat.icon} size={18} />
    </div>
  );
  const meta = (
    <>
      <p class="mt-4 text-3xl font-semibold tracking-tight">{props.stat.value}</p>
      <p class="mt-1 text-sm text-ink-muted">{props.stat.label}</p>
    </>
  );

  return props.stat.href ? (
    <A
      href={props.stat.href}
      class="rounded-xl border border-line bg-surface p-5 transition-colors hover:border-brand"
    >
      <div class="flex items-start justify-between">
        {icon}
        <Icon name="chevron-right" size={15} class="text-ink-faint" />
      </div>
      {meta}
    </A>
  ) : (
    <div class="rounded-xl border border-line bg-surface p-5">
      {icon}
      {meta}
    </div>
  );
}

// Returns a human-readable label for the actor of an audit event.
function actorLabel(e: AuditEvent): string {
  if (e.actor_type === "system") return t("audit.actor.system");
  if (e.actor_name) return e.actor_name;
  return e.actor_type === "token" ? t("audit.actor.token") : currentUser()?.name || "Admin";
}

// Returns an icon name for a given resource type.
function resourceIcon(resourceType: string): IconName {
  if (resourceType === "project") return "folder";
  if (resourceType === "environment") return "globe-alt";
  if (resourceType === "variable") return "lock";
  if (resourceType === "tag") return "tag";
  if (resourceType === "token") return "key";
  return "clock";
}

// Returns a badge variant for a given audit action.
function actionVariant(action: string): "neutral" | "brand" | "success" {
  if (action === "created") return "success";
  if (action === "revealed" || action === "export") return "brand";
  return "neutral";
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
  return new Date(iso).toLocaleDateString(locale(), {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

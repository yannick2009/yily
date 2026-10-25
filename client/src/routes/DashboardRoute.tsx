// Dashboard: at-a-glance overview of projects, environments, secrets and tokens.
import { For, Show, createMemo } from "solid-js";
import { A } from "@solidjs/router";
import { db } from "../store";
import { locale, plural, t } from "../lib/i18n";
import { Card, Icon, type IconName } from "../components/ui";

interface Stat {
  label: string;
  value: number;
  href?: string;
  icon: IconName;
}

export function DashboardRoute() {
  const stats = createMemo<Stat[]>(() => [
    { label: t("nav.projects"), value: db.projects.length, href: "/projects", icon: "folder" },
    {
      label: t("dashboard.environments"),
      value: db.environments.length,
      icon: "globe-alt",
    },
    { label: t("dashboard.secrets"), value: db.variables.length, icon: "lock" },
    { label: t("nav.tokens"), value: db.tokens.length, href: "/tokens", icon: "key" },
  ]);

  const recentVariables = createMemo(() =>
    [...db.variables].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt)).slice(0, 6),
  );

  const projects = createMemo(() =>
    [...db.projects].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt)),
  );

  return (
    <div>
      <div class="mb-6">
        <h1 class="text-2xl font-semibold tracking-tight">{t("dashboard.title")}</h1>
        <p class="mt-1 text-sm text-ink-muted">{t("dashboard.subtitle")}</p>
      </div>

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
              when={recentVariables().length > 0}
              fallback={
                <p class="p-4 text-sm text-ink-muted">{t("dashboard.recentActivityEmpty")}</p>
              }
            >
              <For each={recentVariables()}>
                {(v) => {
                  const env = () => db.environments.find((e) => e.id === v.environmentId);
                  const project = () => {
                    const e = env();
                    return e ? db.projects.find((p) => p.id === e.projectId) : undefined;
                  };
                  const href = () =>
                    `/projects/${project()?.id}/environments/${v.environmentId}/variables/${v.id}`;
                  return (
                    <A
                      href={href()}
                      class="flex items-center gap-3 px-4 py-3 transition-colors hover:bg-surface-2"
                    >
                      <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-surface-2 text-ink-muted">
                        <Icon name="lock" size={15} />
                      </div>
                      <div class="min-w-0 flex-1">
                        <p class="truncate text-sm font-medium text-ink">{v.name}</p>
                        <p class="truncate text-xs text-ink-muted">
                          {[project()?.name, env()?.name].filter(Boolean).join(" · ")}
                        </p>
                      </div>
                      <span class="shrink-0 text-xs text-ink-faint">{timeAgo(v.updatedAt)}</span>
                    </A>
                  );
                }}
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
                {(p) => {
                  const envCount = () =>
                    db.environments.filter((e) => e.projectId === p.id).length;
                  const varCount = () =>
                    db.variables.filter((v) =>
                      db.environments
                        .filter((e) => e.projectId === p.id)
                        .some((e) => e.id === v.environmentId),
                    ).length;
                  return (
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
                          {plural("project.variables", varCount())} ·{" "}
                          {plural("projects.environments", envCount())}
                        </p>
                      </div>
                      <span class="shrink-0 text-xs text-ink-faint">{timeAgo(p.updatedAt)}</span>
                    </A>
                  );
                }}
              </For>
            </Show>
          </Card>
        </section>
      </div>
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

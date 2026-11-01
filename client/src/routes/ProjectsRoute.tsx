// Projects list with create and delete dialogs.
import { createResource, createSignal, For, Show } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { projects } from "../api";
import { plural, t } from "../lib/i18n";
import { Button, Card, Icon, Input } from "../components/ui";
import { Dialog, EmptyState, Field } from "../components/shared";

export function ProjectsRoute() {
  const navigate = useNavigate();
  const [list, { refetch }] = createResource(() => projects.list());

  const [createOpen, setCreateOpen] = createSignal(false);
  const [name, setName] = createSignal("");
  const [error, setError] = createSignal<string | null>(null);
  const [busy, setBusy] = createSignal(false);
  const [deleting, setDeleting] = createSignal<{
    id: string;
    name: string;
  } | null>(null);

  // Navigate to the project detail page when a project is clicked.
  const openProject = (id: string) => navigate(`/projects/${id}`);

  // Create a new project and navigate to its detail page.
  const submitCreate = async (e: Event) => {
    e.preventDefault();
    const n = name().trim();
    if (!n) return;
    setError(null);
    setBusy(true);
    try {
      const created = await projects.create({ name: n });
      setName("");
      setCreateOpen(false);
      await refetch();
      navigate(`/projects/${created.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.delete"));
    } finally {
      setBusy(false);
    }
  };

  const confirmDelete = async () => {
    const d = deleting();
    if (!d) return;
    setBusy(true);
    setError(null);
    try {
      await projects.remove(d.id);
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
          <h1 class="text-2xl font-semibold tracking-tight">{t("nav.projects")}</h1>
          <p class="mt-1 text-sm text-ink-muted">{t("projects.subtitle")}</p>
        </div>
        <Button icon="plus" onClick={() => setCreateOpen(true)}>
          {t("projects.new")}
        </Button>
      </div>

      <Show
        when={!list.loading}
        fallback={<p class="py-8 text-center text-sm text-ink-muted">{t("common.loading")}</p>}
      >
        <Show
          when={(list() ?? []).length > 0}
          fallback={
            <EmptyState
              icon="folder"
              title={t("projects.emptyTitle")}
              description={t("projects.emptyDescription")}
              action={
                <Button icon="plus" onClick={() => setCreateOpen(true)}>
                  {t("projects.new")}
                </Button>
              }
            />
          }
        >
          <ul class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <For each={list() ?? []}>
              {(project) => {
                const envCount = () => project.environment_count ?? 0;
                const tagCount = () => project.tag_count ?? 0;

                return (
                  <li>
                    <Card
                      class="group cursor-pointer p-5 transition-colors hover:border-brand"
                      role="button"
                      tabIndex={0}
                      onClick={() => openProject(project.id)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault();
                          openProject(project.id);
                        }
                      }}
                    >
                      <div class="flex items-start justify-between">
                        <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-subtle text-brand">
                          <Icon name="folder" size={18} />
                        </div>
                        <button
                          type="button"
                          class="rounded-md p-1.5 text-ink-faint opacity-0 transition-opacity hover:bg-danger-subtle hover:text-danger focus-visible:opacity-100 group-hover:opacity-100"
                          onClick={(e) => {
                            e.stopPropagation();
                            setDeleting({ id: project.id, name: project.name });
                          }}
                          aria-label={t("common.deleteAria", {
                            name: project.name,
                          })}
                        >
                          <Icon name="trash" size={15} />
                        </button>
                      </div>

                      <h2 class="mt-4 text-base font-semibold">{project.name}</h2>
                      <div class="mt-2 flex items-center gap-2 text-xs text-ink-muted">
                        <span class="inline-flex items-center gap-1">
                          <Icon name="lock" size={12} />
                          {plural("projects.environments", envCount())}
                        </span>
                        <span aria-hidden="true">·</span>
                        <span>{plural("projects.tags", tagCount())}</span>
                      </div>
                    </Card>
                  </li>
                );
              }}
            </For>
          </ul>
        </Show>
      </Show>

      <Dialog
        open={createOpen()}
        onClose={() => setCreateOpen(false)}
        title={t("projects.new")}
        footer={
          <>
            <Button variant="secondary" onClick={() => setCreateOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" form="new-project-form" loading={busy()}>
              {t("projects.create")}
            </Button>
          </>
        }
      >
        <form id="new-project-form" onSubmit={submitCreate} class="space-y-4">
          <Field label={t("common.name")}>
            <Input
              value={name()}
              onInput={(e) => setName(e.currentTarget.value)}
              placeholder={t("projects.namePlaceholder")}
            />
          </Field>
          <Show when={error()}>
            <p class="text-sm text-danger">{error()}</p>
          </Show>
        </form>
      </Dialog>

      <Dialog
        open={!!deleting()}
        onClose={() => setDeleting(null)}
        title={t("projects.deleteTitle")}
        footer={
          <>
            <Button variant="secondary" onClick={() => setDeleting(null)}>
              {t("common.cancel")}
            </Button>
            <Button variant="danger" onClick={confirmDelete} loading={busy()}>
              {t("common.delete")}
            </Button>
          </>
        }
      >
        <p class="text-sm text-ink-muted">
          {t("projects.deleteConfirm", { name: deleting()?.name ?? "" })}
        </p>
        <Show when={error()}>
          <p class="mt-3 text-sm text-danger">{error()}</p>
        </Show>
      </Dialog>
    </div>
  );
}

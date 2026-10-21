// Projects list with create and delete dialogs.
import { createSignal, For, Show } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { addProject, db, removeProject } from "../store";
import { plural, t } from "../lib/i18n";
import { Button, Card, Icon, Input } from "../components/ui";
import { Dialog, EmptyState, Field } from "../components/shared";

export function ProjectsRoute() {
  const navigate = useNavigate();
  const [createOpen, setCreateOpen] = createSignal(false);
  const [name, setName] = createSignal("");
  const [deleting, setDeleting] = createSignal<{ id: string; name: string } | null>(null);

  const openProject = (id: string) => navigate(`/projects/${id}`);

  const submitCreate = (e: Event) => {
    e.preventDefault();
    const n = name().trim();
    if (!n) return;
    const id = addProject(n);
    setName("");
    setCreateOpen(false);
    navigate(`/projects/${id}`);
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
        when={db.projects.length > 0}
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
          <For each={db.projects}>
            {(project) => {
              const envCount = () => db.environments.filter((e) => e.projectId === project.id).length;
              const tagCount = () => db.tags.filter((t) => t.projectId === project.id).length;

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
                        aria-label={t("common.deleteAria", { name: project.name })}
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
                      <span>
                        {plural("projects.tags", tagCount())}
                      </span>
                    </div>
                  </Card>
                </li>
              );
            }}
          </For>
        </ul>
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
            <Button type="submit" form="new-project-form">
              {t("projects.create")}
            </Button>
          </>
        }
      >
        <form id="new-project-form" onSubmit={submitCreate} class="space-y-4">
          <Field label={t("common.name")}>
            <Input value={name()} onInput={(e) => setName(e.currentTarget.value)} placeholder={t("projects.namePlaceholder")} />
          </Field>
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
            <Button
              variant="danger"
              onClick={() => {
                const d = deleting();
                if (d) {
                  removeProject(d.id);
                  setDeleting(null);
                }
              }}
            >
              {t("common.delete")}
            </Button>
          </>
        }
      >
        <p class="text-sm text-ink-muted">{t("projects.deleteConfirm", { name: deleting()?.name ?? "" })}</p>
      </Dialog>
    </div>
  );
}

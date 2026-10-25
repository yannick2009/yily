// Project detail: inline rename plus environment and tag management.
import { createEffect, createMemo, createSignal, For, Show } from "solid-js";
import { A, useNavigate, useParams } from "@solidjs/router";
import { addEnvironment, addTag, db, removeEnvironment, removeTag, renameProject } from "../store";
import { plural, t } from "../lib/i18n";
import { Badge, Button, Icon, Input, Textarea } from "../components/ui";
import { Dialog, EmptyState, Field } from "../components/shared";

export function ProjectDetailRoute() {
  const params = useParams();
  const navigate = useNavigate();
  const projectId = () => params.projectId ?? "";

  const project = createMemo(() => db.projects.find((p) => p.id === projectId()));
  const environments = createMemo(() => db.environments.filter((e) => e.projectId === projectId()));
  const tags = createMemo(() => db.tags.filter((t) => t.projectId === projectId()));

  const varCount = (envId: string) => db.variables.filter((v) => v.environmentId === envId).length;
  const openEnv = (envId: string) => navigate(`/projects/${projectId()}/environments/${envId}`);

  // Rename project.
  let renameInput: HTMLInputElement | undefined;
  const [renaming, setRenaming] = createSignal(false);
  const [draft, setDraft] = createSignal("");

  createEffect(() => {
    if (renaming()) renameInput?.focus();
  });

  const startRename = () => {
    setDraft(project()?.name ?? "");
    setRenaming(true);
  };
  const commitRename = () => {
    const n = draft().trim();
    if (n && project()) renameProject(projectId(), n);
    setRenaming(false);
  };
  const onRenameKey = (e: KeyboardEvent) => {
    if (e.key === "Enter") commitRename();
    else if (e.key === "Escape") setRenaming(false);
  };

  // Create environment.
  const [envOpen, setEnvOpen] = createSignal(false);
  const [envName, setEnvName] = createSignal("");
  const [envDesc, setEnvDesc] = createSignal("");
  const submitEnv = (e: Event) => {
    e.preventDefault();
    const n = envName().trim();
    if (!n) return;
    addEnvironment(projectId(), n, envDesc().trim());
    setEnvName("");
    setEnvDesc("");
    setEnvOpen(false);
  };

  // Create tag.
  const [tagOpen, setTagOpen] = createSignal(false);
  const [tagName, setTagName] = createSignal("");
  const submitTag = (e: Event) => {
    e.preventDefault();
    const n = tagName().trim();
    if (!n) return;
    addTag(projectId(), n);
    setTagName("");
    setTagOpen(false);
  };

  // Deletions.
  const [deletingEnv, setDeletingEnv] = createSignal<{ id: string; name: string } | null>(null);
  const [deletingTag, setDeletingTag] = createSignal<{ id: string; name: string } | null>(null);

  return (
    <Show
      when={project()}
      fallback={
        <div class="space-y-4">
          <p class="text-sm text-ink-muted">{t("project.notFound")}</p>
          <A
            href="/projects"
            class="inline-flex items-center gap-1.5 text-sm text-brand hover:underline"
          >
            <Icon name="arrow-left" size={14} /> {t("common.backToProjects")}
          </A>
        </div>
      }
    >
      <div>
        <A
          href="/projects"
          class="mb-3 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
        >
          <Icon name="arrow-left" size={14} /> {t("nav.projects")}
        </A>

        <div class="mb-6">
          <Show
            when={renaming()}
            fallback={
              <div class="flex items-center gap-2">
                <h1 class="text-2xl font-semibold tracking-tight">{project()!.name}</h1>
                <button
                  type="button"
                  onClick={startRename}
                  class="rounded-md p-1.5 text-ink-faint hover:bg-surface-2 hover:text-ink"
                  aria-label={t("project.renameAria")}
                >
                  <Icon name="pencil" size={15} />
                </button>
              </div>
            }
          >
            <Input
              ref={(el) => (renameInput = el)}
              value={draft()}
              onInput={(e) => setDraft(e.currentTarget.value)}
              onKeyDown={onRenameKey}
              onBlur={commitRename}
              class="max-w-xs"
            />
          </Show>
        </div>

        <div class="grid gap-6 lg:grid-cols-2">
          <section>
            <div class="mb-3 flex items-center justify-between">
              <h2 class="text-base font-semibold">{t("project.environments")}</h2>
              <Button size="sm" variant="secondary" icon="plus" onClick={() => setEnvOpen(true)}>
                {t("project.newEnvironment")}
              </Button>
            </div>

            <Show
              when={environments().length > 0}
              fallback={
                <EmptyState
                  icon="lock"
                  title={t("project.noEnvironments")}
                  description={t("project.noEnvironmentsDesc")}
                />
              }
            >
              <ul class="space-y-2">
                <For each={environments()}>
                  {(env) => (
                    <li>
                      <div
                        class="group flex cursor-pointer items-center justify-between gap-3 rounded-lg border border-line bg-surface px-4 py-3 transition-colors hover:border-brand"
                        role="button"
                        tabIndex={0}
                        onClick={() => openEnv(env.id)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter" || e.key === " ") {
                            e.preventDefault();
                            openEnv(env.id);
                          }
                        }}
                      >
                        <div class="flex min-w-0 flex-1 items-center gap-3">
                          <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-surface-2 text-ink-muted">
                            <Icon name="lock" size={16} />
                          </div>
                          <div class="min-w-0 flex-1">
                            <p class="truncate text-sm font-medium">{env.name}</p>
                            <p class="truncate text-xs text-ink-muted" title={env.description}>
                              {env.description || t("project.noDescription")}
                            </p>
                          </div>
                        </div>

                        <div class="flex shrink-0 items-center gap-1">
                          <span class="mr-2 text-xs text-ink-faint">
                            {plural("project.variables", varCount(env.id))}
                          </span>
                          <button
                            type="button"
                            class="rounded-md p-1.5 text-ink-faint opacity-0 transition-opacity hover:bg-danger-subtle hover:text-danger focus-visible:opacity-100 group-hover:opacity-100"
                            onClick={(e) => {
                              e.stopPropagation();
                              setDeletingEnv({ id: env.id, name: env.name });
                            }}
                            aria-label={t("common.deleteAria", { name: env.name })}
                          >
                            <Icon name="trash" size={15} />
                          </button>
                          <Icon name="chevron-right" size={16} class="text-ink-faint" />
                        </div>
                      </div>
                    </li>
                  )}
                </For>
              </ul>
            </Show>
          </section>

          <section>
            <div class="mb-3 flex items-center justify-between">
              <h2 class="text-base font-semibold">{t("project.tags")}</h2>
              <Button size="sm" variant="secondary" icon="plus" onClick={() => setTagOpen(true)}>
                {t("project.newTag")}
              </Button>
            </div>

            <Show
              when={tags().length > 0}
              fallback={
                <EmptyState
                  icon="tag"
                  title={t("project.noTags")}
                  description={t("project.noTagsDesc")}
                />
              }
            >
              <ul class="space-y-2">
                <For each={tags()}>
                  {(tag) => (
                    <li class="flex items-center justify-between rounded-lg border border-line bg-surface px-3 py-2">
                      <Badge variant="brand">
                        <Icon name="tag" size={12} />
                        {tag.name}
                      </Badge>
                      <button
                        type="button"
                        onClick={() => setDeletingTag({ id: tag.id, name: tag.name })}
                        class="rounded p-1 text-ink-faint hover:bg-danger-subtle hover:text-danger"
                        aria-label={t("project.deleteTagAria", { name: tag.name })}
                      >
                        <Icon name="x" size={14} />
                      </button>
                    </li>
                  )}
                </For>
              </ul>
            </Show>
          </section>
        </div>
      </div>

      <Dialog
        open={envOpen()}
        onClose={() => setEnvOpen(false)}
        title={t("project.newEnvironment")}
        footer={
          <>
            <Button variant="secondary" onClick={() => setEnvOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" form="new-env-form">
              {t("project.createEnvironment")}
            </Button>
          </>
        }
      >
        <form id="new-env-form" onSubmit={submitEnv} class="space-y-4">
          <Field label={t("common.name")}>
            <Input
              value={envName()}
              onInput={(e) => setEnvName(e.currentTarget.value)}
              placeholder={t("project.envNamePlaceholder")}
            />
          </Field>
          <Field label={t("common.description")} hint={t("common.optional")}>
            <Textarea
              rows={2}
              value={envDesc()}
              onInput={(e) => setEnvDesc(e.currentTarget.value)}
              placeholder={t("project.envDescPlaceholder")}
            />
          </Field>
        </form>
      </Dialog>

      <Dialog
        open={tagOpen()}
        onClose={() => setTagOpen(false)}
        title={t("project.newTag")}
        footer={
          <>
            <Button variant="secondary" onClick={() => setTagOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" form="new-tag-form">
              {t("project.createTag")}
            </Button>
          </>
        }
      >
        <form id="new-tag-form" onSubmit={submitTag} class="space-y-4">
          <Field label={t("common.name")}>
            <Input
              value={tagName()}
              onInput={(e) => setTagName(e.currentTarget.value)}
              placeholder={t("project.tagNamePlaceholder")}
            />
          </Field>
        </form>
      </Dialog>

      <Dialog
        open={!!deletingEnv()}
        onClose={() => setDeletingEnv(null)}
        title={t("project.deleteEnvironment")}
        footer={
          <>
            <Button variant="secondary" onClick={() => setDeletingEnv(null)}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="danger"
              onClick={() => {
                const d = deletingEnv();
                if (d) {
                  removeEnvironment(d.id);
                  setDeletingEnv(null);
                }
              }}
            >
              {t("common.delete")}
            </Button>
          </>
        }
      >
        <p class="text-sm text-ink-muted">
          {t("project.deleteEnvironmentConfirm", { name: deletingEnv()?.name ?? "" })}
        </p>
      </Dialog>

      <Dialog
        open={!!deletingTag()}
        onClose={() => setDeletingTag(null)}
        title={t("project.deleteTag")}
        footer={
          <>
            <Button variant="secondary" onClick={() => setDeletingTag(null)}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="danger"
              onClick={() => {
                const d = deletingTag();
                if (d) {
                  removeTag(projectId(), d.id);
                  setDeletingTag(null);
                }
              }}
            >
              {t("common.delete")}
            </Button>
          </>
        }
      >
        <p class="text-sm text-ink-muted">
          {t("project.deleteTagConfirm", { name: deletingTag()?.name ?? "" })}
        </p>
      </Dialog>
    </Show>
  );
}

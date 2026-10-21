// Variable detail: current value, version history with activate/deactivate, and tag editing.
import { createEffect, createMemo, createSignal, For, Show } from "solid-js";
import { A, useParams } from "@solidjs/router";
import {
  addVariableVersion,
  clearVariableValue,
  db,
  renameVariable,
  setVariableTags,
  setVariableVersionActive,
} from "../store";
import { locale, t } from "../lib/i18n";
import { Badge, Button, Card, Icon, Input, Textarea } from "../components/ui";
import { Dialog, EmptyState, Field, ValueGenerator } from "../components/shared";

export function VariableDetailRoute() {
  const params = useParams();
  const projectId = () => params.projectId ?? "";
  const envId = () => params.envId ?? "";
  const variableId = () => params.variableId ?? "";

  const variable = createMemo(() => db.variables.find((v) => v.id === variableId()));
  const environment = createMemo(() => db.environments.find((e) => e.id === envId()));

  // Newest first, mirroring GET /variables/:id/versions.
  const versions = createMemo(() =>
    db.variableVersions
      .filter((v) => v.variableId === variableId())
      .sort((a, b) => b.version - a.version),
  );
  const currentVersion = createMemo(() => versions().find((v) => v.active) ?? null);
  const variableTags = createMemo(() =>
    (variable()?.tagIds ?? []).map((id) => db.tags.find((t) => t.id === id)),
  );
  const projectTags = createMemo(() => db.tags.filter((t) => t.projectId === projectId()));

  // Reveal + copy per version.
  const [revealedIds, setRevealedIds] = createSignal<string[]>([]);
  const [copiedId, setCopiedId] = createSignal<string | null>(null);
  const isRevealed = (id: string) => revealedIds().includes(id);
  const toggleReveal = (id: string) =>
    setRevealedIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));
  const copy = async (id: string, value: string) => {
    try {
      await navigator.clipboard.writeText(value);
    } catch {
      /* clipboard unavailable */
    }
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 1500);
  };

  // New version.
  const [newOpen, setNewOpen] = createSignal(false);
  const [value, setValue] = createSignal("");
  const submitNew = (e: Event) => {
    e.preventDefault();
    const v = value().trim();
    if (!v) return;
    addVariableVersion(variableId(), v);
    setValue("");
    setNewOpen(false);
  };

  // Rename variable.
  let renameInput: HTMLInputElement | undefined;
  const [renaming, setRenaming] = createSignal(false);
  const [draft, setDraft] = createSignal("");

  createEffect(() => {
    if (renaming()) renameInput?.focus();
  });

  const startRename = () => {
    setDraft(variable()?.name ?? "");
    setRenaming(true);
  };
  const commitRename = () => {
    const n = draft().trim();
    if (n && variable()) renameVariable(variableId(), n);
    setRenaming(false);
  };
  const onRenameKey = (e: KeyboardEvent) => {
    if (e.key === "Enter") commitRename();
    else if (e.key === "Escape") setRenaming(false);
  };

  // Edit tags.
  const [tagsOpen, setTagsOpen] = createSignal(false);
  const [selectedTagIds, setSelectedTagIds] = createSignal<string[]>([]);
  const openTags = () => {
    setSelectedTagIds(variable()?.tagIds ?? []);
    setTagsOpen(true);
  };
  const toggleTag = (id: string) =>
    setSelectedTagIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));
  const saveTags = () => {
    setVariableTags(variableId(), selectedTagIds());
    setTagsOpen(false);
  };

  // Clear value.
  const [clearOpen, setClearOpen] = createSignal(false);
  const clearValue = () => {
    clearVariableValue(variableId());
    setClearOpen(false);
  };

  return (
    <Show
      when={variable()}
      fallback={
        <div class="space-y-4">
          <p class="text-sm text-ink-muted">{t("variable.notFound")}</p>
          <A
            href={`/projects/${projectId()}/environments/${envId()}`}
            class="inline-flex items-center gap-1.5 text-sm text-brand hover:underline"
          >
            <Icon name="arrow-left" size={14} /> {environment()?.name ?? ""}
          </A>
        </div>
      }
    >
      <div>
        <A
          href={`/projects/${projectId()}/environments/${envId()}`}
          class="mb-3 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
        >
          <Icon name="arrow-left" size={14} /> {environment()?.name ?? ""}
        </A>

        <div class="mb-6 flex items-start justify-between gap-4">
          <div class="min-w-0">
            <Show
              when={renaming()}
              fallback={
                <div class="flex items-center gap-2">
                  <h1 class="font-mono text-2xl font-semibold tracking-tight">
                    {variable()!.name}
                  </h1>
                  <button
                    type="button"
                    onClick={startRename}
                    class="rounded-md p-1.5 text-ink-faint hover:bg-surface-2 hover:text-ink"
                    aria-label={t("variable.renameAria")}
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
                class="max-w-xs font-mono"
              />
            </Show>
            <div class="mt-2 flex flex-wrap items-center gap-1">
              <For each={variableTags()}>
                {(tag) => (tag ? <Badge variant="brand">{tag.name}</Badge> : null)}
              </For>
              <button
                type="button"
                onClick={openTags}
                class="rounded-md p-1 text-ink-faint hover:bg-surface-2 hover:text-ink"
                aria-label={t("variable.editTags")}
              >
                <Icon name="pencil" size={13} />
              </button>
            </div>
            <p class="mt-2 text-xs text-ink-muted">
              {t("variable.created", { date: formatDate(variable()!.createdAt) })}
            </p>
          </div>
          <Button icon="plus" onClick={() => setNewOpen(true)}>
            {t("variable.newVersion")}
          </Button>
        </div>

        <section class="mb-6 rounded-xl border border-line bg-surface p-4">
          <div class="mb-2 flex items-center justify-between gap-2">
            <h2 class="text-sm font-semibold text-ink">{t("variable.currentValue")}</h2>
            <div class="flex items-center gap-2">
              <Show when={currentVersion()}>
                <Badge variant="brand">
                  {t("variable.version", { n: currentVersion()!.version })}
                </Badge>
              </Show>
              <Show when={currentVersion()}>
                <Button size="sm" variant="ghost" onClick={() => setClearOpen(true)}>
                  {t("variable.clearValue")}
                </Button>
              </Show>
            </div>
          </div>
          <Show
            when={currentVersion()}
            fallback={<p class="text-sm text-ink-muted">{t("variable.noActiveValue")}</p>}
          >
            <div class="flex items-start gap-2">
              <code class="min-w-0 flex-1 break-all font-mono text-sm text-ink-muted">
                {isRevealed(currentVersion()!.id) ? currentVersion()!.value : "••••••••••"}
              </code>
              <div class="flex shrink-0 items-center gap-1">
                <button
                  type="button"
                  onClick={() => toggleReveal(currentVersion()!.id)}
                  class="rounded-md p-1.5 text-ink-muted hover:bg-surface-2 hover:text-ink"
                  aria-label={
                    isRevealed(currentVersion()!.id)
                      ? t("variable.hideAria", { n: currentVersion()!.version })
                      : t("variable.revealAria", { n: currentVersion()!.version })
                  }
                >
                  <Icon name={isRevealed(currentVersion()!.id) ? "eye-off" : "eye"} size={15} />
                </button>
                <button
                  type="button"
                  onClick={() => copy(currentVersion()!.id, currentVersion()!.value)}
                  class="rounded-md p-1.5 text-ink-muted hover:bg-surface-2 hover:text-ink"
                  aria-label={t("variable.copyAria", { n: currentVersion()!.version })}
                >
                  <Icon name={copiedId() === currentVersion()!.id ? "check" : "copy"} size={15} />
                </button>
              </div>
            </div>
          </Show>
        </section>

        <section>
          <h2 class="mb-3 text-base font-semibold">{t("variable.versions")}</h2>
          <Show
            when={versions().length > 0}
            fallback={
              <EmptyState
                icon="lock"
                title={t("variable.noVersions")}
                description={t("variable.noVersionsDesc")}
              />
            }
          >
            <Card class="divide-y divide-line">
              <For each={versions()}>
                {(vv) => {
                  const revealed = () => isRevealed(vv.id);
                  return (
                    <div class="flex items-center gap-4 px-4 py-3.5">
                      <div class="min-w-0 flex-1">
                        <div class="flex items-center gap-2">
                          <span class="text-sm font-medium text-ink">
                            {t("variable.version", { n: vv.version })}
                          </span>
                          {vv.active ? (
                            <Badge variant="brand">{t("variable.active")}</Badge>
                          ) : (
                            <Badge>{t("variable.inactive")}</Badge>
                          )}
                        </div>
                        <p class="mt-0.5 text-xs text-ink-muted">
                          {t("variable.created", { date: formatDate(vv.createdAt) })}
                        </p>
                        <code class="mt-1 block break-all font-mono text-sm text-ink-muted">
                          {revealed() ? vv.value : "••••••••••"}
                        </code>
                      </div>
                      <div class="flex shrink-0 items-center gap-1">
                        <button
                          type="button"
                          onClick={() => toggleReveal(vv.id)}
                          class="rounded-md p-1.5 text-ink-muted hover:bg-surface-2 hover:text-ink"
                          aria-label={
                            revealed()
                              ? t("variable.hideAria", { n: vv.version })
                              : t("variable.revealAria", { n: vv.version })
                          }
                        >
                          <Icon name={revealed() ? "eye-off" : "eye"} size={15} />
                        </button>
                        <button
                          type="button"
                          onClick={() => copy(vv.id, vv.value)}
                          class="rounded-md p-1.5 text-ink-muted hover:bg-surface-2 hover:text-ink"
                          aria-label={t("variable.copyAria", { n: vv.version })}
                        >
                          <Icon name={copiedId() === vv.id ? "check" : "copy"} size={15} />
                        </button>
                        <Button
                          size="sm"
                          variant={vv.active ? "secondary" : "primary"}
                          onClick={() => setVariableVersionActive(vv.id, !vv.active)}
                        >
                          {vv.active ? t("variable.deactivate") : t("variable.activate")}
                        </Button>
                      </div>
                    </div>
                  );
                }}
              </For>
            </Card>
          </Show>
        </section>
      </div>

      <Dialog
        open={newOpen()}
        onClose={() => setNewOpen(false)}
        title={t("variable.newVersion")}
        footer={
          <>
            <Button variant="secondary" onClick={() => setNewOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" form="new-version-form">
              {t("variable.newVersion")}
            </Button>
          </>
        }
      >
        <form id="new-version-form" onSubmit={submitNew} class="space-y-4">
          <Field label={t("env.value")} hint={t("env.encryptedHint")}>
            <Textarea
              rows={3}
              value={value()}
              onInput={(e) => setValue(e.currentTarget.value)}
              placeholder={t("variable.valuePlaceholder")}
            />
            <ValueGenerator onGenerate={(v) => setValue(v)} />
          </Field>
        </form>
      </Dialog>

      <Dialog
        open={tagsOpen()}
        onClose={() => setTagsOpen(false)}
        title={t("variable.editTags")}
        footer={
          <>
            <Button variant="secondary" onClick={() => setTagsOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={saveTags}>{t("common.save")}</Button>
          </>
        }
      >
        <Show
          when={projectTags().length > 0}
          fallback={<p class="text-sm text-ink-muted">{t("project.noTags")}</p>}
        >
          <ul class="space-y-2">
            <For each={projectTags()}>
              {(tag) => {
                const checked = () => selectedTagIds().includes(tag.id);
                return (
                  <li>
                    <label class="flex cursor-pointer items-center gap-2 text-sm text-ink">
                      <input
                        type="checkbox"
                        checked={checked()}
                        onChange={() => toggleTag(tag.id)}
                        class="h-4 w-4 rounded border-line accent-brand"
                      />
                      {tag.name}
                    </label>
                  </li>
                );
              }}
            </For>
          </ul>
        </Show>
      </Dialog>

      <Dialog
        open={clearOpen()}
        onClose={() => setClearOpen(false)}
        title={t("variable.clearValue")}
        footer={
          <>
            <Button variant="secondary" onClick={() => setClearOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button variant="danger" onClick={clearValue}>
              {t("variable.clearValue")}
            </Button>
          </>
        }
      >
        <p class="text-sm text-ink-muted">
          {t("variable.clearValueConfirm", { name: variable()?.name ?? "" })}
        </p>
      </Dialog>
    </Show>
  );
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(locale(), {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

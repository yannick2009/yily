// Variables table for an environment: tag filter, value reveal/copy, and file import/export.
import { createEffect, createResource, createSignal, For, Show, onCleanup } from "solid-js";
import { A, useParams } from "@solidjs/router";
import { environments, projects, tags, variables } from "../api";
import { t } from "../lib/i18n";
import { createRevealController } from "../lib/reveal";
import { Badge, Button, Icon, Input, Textarea } from "../components/ui";
import { Dialog, EmptyState, Field, RevealCountdown, ValueGenerator } from "../components/shared";

export function EnvironmentRoute() {
  const params = useParams();
  const projectId = () => params.projectId ?? "";
  const envId = () => params.envId ?? "";

  // Tag filter (matches at least one selected tag), applied server-side.
  const [selectedTags, setSelectedTags] = createSignal<string[]>([]);
  const toggleTag = (id: string) =>
    setSelectedTags((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));

  // Text search (by variable name), debounced and applied server-side.
  const [query, setQuery] = createSignal("");
  const [debouncedQuery, setDebouncedQuery] = createSignal("");
  let searchTimer: ReturnType<typeof setTimeout> | undefined;
  const onSearchInput = (e: Event) => {
    const value = (e.currentTarget as HTMLInputElement).value;
    setQuery(value);
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => setDebouncedQuery(value), 250);
  };
  onCleanup(() => clearTimeout(searchTimer));

  const [envRes, { refetch: refetchEnv }] = createResource(() => environments.get(envId()));
  const [projectRes] = createResource(() => projects.get(projectId()));
  const [varsRes, { refetch: refetchVars }] = createResource(
    () => ({ envId: envId(), tagIds: selectedTags(), search: debouncedQuery() }),
    ({ envId, tagIds, search }) => variables.list(envId, { tagIds, search }),
  );
  const [tagsRes] = createResource(() => tags.list(projectId()));

  const environment = () => envRes();
  const variableList = () => varsRes() ?? [];
  const tagList = () => tagsRes() ?? [];

  // Whether the current list is narrowed by a search and/or tag filter.
  const filtering = () => debouncedQuery().trim() !== "" || selectedTags().length > 0;
  // Whether the environment has variables at all, even when the filtered list is empty.
  const hasVariables = () => variableList().length > 0 || filtering();

  // Value reveal + copy. Revealed values auto-hide after 20s, with a countdown.
  // The server only exposes the current value on demand, so we fetch it here.
  const reveal = createRevealController();
  const [revealedValues, setRevealedValues] = createSignal<Record<string, string>>({});
  const [copiedId, setCopiedId] = createSignal<string | null>(null);

  const toggleReveal = async (id: string) => {
    if (reveal.isRevealed(id)) {
      reveal.hide(id);
      setRevealedValues((prev) => {
        const next = { ...prev };
        delete next[id];
        return next;
      });
      return;
    }
    try {
      const value = await variables.getValue(id);
      setRevealedValues((prev) => ({ ...prev, [id]: value }));
      reveal.show(id);
    } catch {
      /* reveal failed; leave hidden */
    }
  };

  const copy = async (id: string) => {
    try {
      await navigator.clipboard.writeText(revealedValues()[id] ?? "");
    } catch {
      /* clipboard unavailable */
    }
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 1500);
  };

  // Create variable.
  const [createOpen, setCreateOpen] = createSignal(false);
  const [name, setName] = createSignal("");
  const [value, setValue] = createSignal("");
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const submitCreate = async (e: Event) => {
    e.preventDefault();
    const n = name().trim();
    if (!n) return;
    setBusy(true);
    setError(null);
    try {
      await variables.create(envId(), { name: n, value: value() });
      setName("");
      setValue("");
      setCreateOpen(false);
      await refetchVars();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.save"));
    } finally {
      setBusy(false);
    }
  };

  // Delete variable.
  const [deleting, setDeleting] = createSignal<{ id: string; name: string } | null>(null);
  const confirmDelete = async () => {
    const d = deleting();
    if (!d) return;
    setBusy(true);
    setError(null);
    try {
      await variables.remove(d.id);
      setDeleting(null);
      await refetchVars();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.delete"));
    } finally {
      setBusy(false);
    }
  };

  // Rename environment.
  let renameInput: HTMLInputElement | undefined;
  const [renaming, setRenaming] = createSignal(false);
  const [draftName, setDraftName] = createSignal("");

  createEffect(() => {
    if (renaming()) renameInput?.focus();
  });

  const startRename = () => {
    setDraftName(environment()?.name ?? "");
    setRenaming(true);
  };
  const commitRename = async () => {
    const n = draftName().trim();
    if (n && environment()) {
      try {
        await environments.rename(envId(), n);
        await refetchEnv();
      } catch (err) {
        setError(err instanceof Error ? err.message : t("common.save"));
      }
    }
    setRenaming(false);
  };
  const onRenameKey = (e: KeyboardEvent) => {
    if (e.key === "Enter") void commitRename();
    else if (e.key === "Escape") setRenaming(false);
  };

  // Edit description.
  let descInput: HTMLInputElement | undefined;
  const [editingDesc, setEditingDesc] = createSignal(false);
  const [draftDesc, setDraftDesc] = createSignal("");

  createEffect(() => {
    if (editingDesc()) descInput?.focus();
  });

  const startDescEdit = () => {
    setDraftDesc(environment()?.description ?? "");
    setEditingDesc(true);
  };
  const commitDesc = async () => {
    const d = draftDesc().trim();
    if (environment()) {
      try {
        await environments.setDescription(envId(), d);
        await refetchEnv();
      } catch (err) {
        setError(err instanceof Error ? err.message : t("common.save"));
      }
    }
    setEditingDesc(false);
  };
  const onDescKey = (e: KeyboardEvent) => {
    if (e.key === "Enter") void commitDesc();
    else if (e.key === "Escape") setEditingDesc(false);
  };

  // Import.
  let fileInput: HTMLInputElement | undefined;
  const [importOpen, setImportOpen] = createSignal(false);
  const [importError, setImportError] = createSignal<string | null>(null);
  const [dragging, setDragging] = createSignal(false);
  const [importBusy, setImportBusy] = createSignal(false);

  createEffect(() => {
    if (importOpen()) {
      queueMicrotask(() => {
        if (fileInput) fileInput.value = "";
      });
      setDragging(false);
    }
  });

  const openImport = () => {
    setImportError(null);
    setImportOpen(true);
  };
  const closeImport = () => {
    setImportOpen(false);
    setImportError(null);
    setDragging(false);
  };

  const handleFile = async (file: File) => {
    setImportBusy(true);
    setImportError(null);
    try {
      await environments.importFile(envId(), file);
      setImportOpen(false);
      await refetchVars();
    } catch (err) {
      setImportError(err instanceof Error ? err.message : t("env.importInvalid"));
    } finally {
      setImportBusy(false);
    }
  };

  const onImportFile = (e: Event) => {
    const input = e.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (file) void handleFile(file);
  };

  const onDragOver = (e: DragEvent) => {
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = "copy";
    setDragging(true);
  };
  const onDragLeave = () => setDragging(false);
  const onDrop = (e: DragEvent) => {
    e.preventDefault();
    setDragging(false);
    const file = e.dataTransfer?.files?.[0];
    if (file) void handleFile(file);
  };

  // Export.
  const [exportOpen, setExportOpen] = createSignal(false);
  const [exportBusy, setExportBusy] = createSignal(false);

  const doExport = async (format: FormatOption) => {
    setExportBusy(true);
    setError(null);
    try {
      const { blob, filename } = await environments.exportFile(envId(), format.id);
      downloadBlob(blob, filename);
      setExportOpen(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.save"));
    } finally {
      setExportBusy(false);
    }
  };

  return (
    <Show
      when={!envRes.loading}
      fallback={<p class="py-8 text-center text-sm text-ink-muted">{t("common.loading")}</p>}
    >
      <Show
        when={environment()}
        fallback={
          <div class="space-y-4">
            <p class="text-sm text-ink-muted">{t("env.notFound")}</p>
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
            href={`/projects/${projectId()}`}
            class="mb-3 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
          >
            <Icon name="arrow-left" size={14} /> {projectRes()?.name ?? ""}
          </A>

          <div class="mb-6 flex items-start justify-between gap-4">
            <div class="min-w-0">
              <Show
                when={renaming()}
                fallback={
                  <div class="flex items-center gap-2">
                    <h1 class="text-2xl font-semibold tracking-tight">{environment()!.name}</h1>
                    <button
                      type="button"
                      onClick={startRename}
                      class="rounded-md p-1.5 text-ink-faint hover:bg-surface-2 hover:text-ink"
                      aria-label={t("env.renameAria")}
                    >
                      <Icon name="pencil" size={15} />
                    </button>
                  </div>
                }
              >
                <Input
                  ref={(el) => (renameInput = el)}
                  value={draftName()}
                  onInput={(e) => setDraftName(e.currentTarget.value)}
                  onKeyDown={onRenameKey}
                  onBlur={commitRename}
                  class="max-w-xs"
                />
              </Show>

              <Show
                when={editingDesc()}
                fallback={
                  <div class="mt-1 flex items-center gap-2">
                    <p class="text-sm text-ink-muted">
                      {environment()!.description || t("project.noDescription")}
                    </p>
                    <button
                      type="button"
                      onClick={startDescEdit}
                      class="rounded-md p-1 text-ink-faint hover:bg-surface-2 hover:text-ink"
                      aria-label={t("env.editDescriptionAria")}
                    >
                      <Icon name="pencil" size={13} />
                    </button>
                  </div>
                }
              >
                <Input
                  ref={(el) => (descInput = el)}
                  value={draftDesc()}
                  onInput={(e) => setDraftDesc(e.currentTarget.value)}
                  onKeyDown={onDescKey}
                  onBlur={commitDesc}
                  class="mt-1 max-w-xs"
                  placeholder={t("project.envDescPlaceholder")}
                />
              </Show>
            </div>
            <div class="flex gap-2">
              <Button variant="secondary" icon="download" onClick={openImport}>
                {t("env.import")}
              </Button>
              <Button variant="secondary" icon="upload" onClick={() => setExportOpen(true)}>
                {t("env.export")}
              </Button>
              <Button icon="plus" onClick={() => setCreateOpen(true)}>
                {t("env.newVariable")}
              </Button>
            </div>
          </div>

          <Show when={hasVariables()}>
            <div class="mb-4 space-y-3">
              <div class="relative w-full sm:w-72">
                <Input
                  value={query()}
                  onInput={onSearchInput}
                  placeholder={t("env.searchPlaceholder")}
                  style={{ "padding-left": "2.5rem" }}
                />
                <span class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-ink-faint">
                  <Icon name="search" size={16} />
                </span>
              </div>
              <Show when={tagList().length > 0}>
                <div class="flex flex-wrap items-center gap-2">
                  <span class="mr-1 text-xs font-medium uppercase tracking-wide text-ink-faint">
                    {t("env.filter")}
                  </span>
                  <For each={tagList()}>
                    {(tag) => {
                      const active = () => selectedTags().includes(tag.id);
                      return (
                        <button
                          type="button"
                          onClick={() => toggleTag(tag.id)}
                          class={[
                            "rounded-full px-3 py-1 text-xs font-medium transition-colors",
                            active()
                              ? "bg-brand text-on-brand"
                              : "bg-surface-2 text-ink-muted hover:text-ink",
                          ].join(" ")}
                        >
                          {tag.name}
                        </button>
                      );
                    }}
                  </For>
                </div>
              </Show>
            </div>
          </Show>

          <Show
            when={!varsRes.loading}
            fallback={<p class="py-8 text-center text-sm text-ink-muted">{t("common.loading")}</p>}
          >
            <Show
              when={variableList().length > 0}
              fallback={
                <EmptyState
                  icon="lock"
                  title={filtering() ? t("env.noVariablesMatch") : t("env.noVariables")}
                  description={filtering() ? t("env.tryRemoving") : t("env.addFirst")}
                  action={
                    filtering() ? undefined : (
                      <Button icon="plus" onClick={() => setCreateOpen(true)}>
                        {t("env.newVariable")}
                      </Button>
                    )
                  }
                />
              }
            >
              <div class="overflow-x-auto rounded-xl border border-line bg-surface">
                <table class="w-full table-fixed border-collapse text-left text-sm">
                  <thead>
                    <tr class="border-b border-line text-xs uppercase tracking-wide text-ink-faint">
                      <th class="px-4 py-3 font-medium">{t("common.name")}</th>
                      <th class="w-16 px-4 py-3 text-center font-medium">{t("env.status")}</th>
                      <th class="w-64 px-4 py-3 font-medium">{t("env.value")}</th>
                      <th class="px-4 py-3 font-medium">{t("project.tags")}</th>
                      <th class="w-32 px-4 py-3 text-right font-medium">{t("env.actions")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <For each={variableList()}>
                      {(v) => {
                        const revealed = () => reveal.isRevealed(v.id);
                        const remaining = () => reveal.countdownFor(v.id);
                        const currentValue = () => revealedValues()[v.id] ?? "";
                        const variableTags = () => v.tags ?? [];

                        return (
                          <tr class="border-t border-line first:border-t-0">
                            <td class="py-3 pl-4 pr-4 align-top">
                              <A
                                href={`/projects/${projectId()}/environments/${envId()}/variables/${v.id}`}
                                class="font-mono text-sm text-ink hover:text-brand hover:underline"
                              >
                                {v.name}
                              </A>
                            </td>
                            <td class="py-3 pl-4 pr-4 text-center align-top">
                              <span
                                class={[
                                  "mt-1.5 inline-block h-1.5 w-1.5 rounded-full",
                                  v.active ? "bg-success" : "bg-ink-faint",
                                ].join(" ")}
                                title={v.active ? t("variable.active") : t("variable.inactive")}
                              />
                            </td>
                            <td class="py-3 pr-4 align-top">
                              <div class="flex items-center gap-2">
                                <code
                                  class="min-w-0 flex-1 truncate font-mono text-sm text-ink-muted"
                                  title={revealed() ? currentValue() || undefined : undefined}
                                >
                                  {revealed()
                                    ? truncateMiddle(currentValue() || t("env.noValue"))
                                    : "••••••••••"}
                                </code>
                                <Show when={revealed() && remaining() > 0}>
                                  <RevealCountdown seconds={remaining()} />
                                </Show>
                              </div>
                            </td>
                            <td class="py-3 pr-4 align-top">
                              <div class="flex flex-wrap gap-1">
                                <For each={variableTags()}>
                                  {(tag) => <Badge variant="brand">{tag.name}</Badge>}
                                </For>
                              </div>
                            </td>
                            <td class="py-3 pr-4 text-right align-top">
                              <div class="flex items-center justify-end gap-1">
                                <button
                                  type="button"
                                  onClick={() => toggleReveal(v.id)}
                                  class="rounded-md p-1.5 text-ink-muted hover:bg-surface-2 hover:text-ink"
                                  aria-label={
                                    revealed()
                                      ? t("env.hideAria", { name: v.name })
                                      : t("env.revealAria", { name: v.name })
                                  }
                                >
                                  <Icon name={revealed() ? "eye-off" : "eye"} size={15} />
                                </button>
                                <button
                                  type="button"
                                  onClick={() => copy(v.id)}
                                  class="rounded-md p-1.5 text-ink-muted hover:bg-surface-2 hover:text-ink"
                                  aria-label={t("env.copyAria", { name: v.name })}
                                >
                                  <Icon name={copiedId() === v.id ? "check" : "copy"} size={15} />
                                </button>
                                <button
                                  type="button"
                                  onClick={() => setDeleting({ id: v.id, name: v.name })}
                                  class="rounded-md p-1.5 text-ink-muted hover:bg-danger-subtle hover:text-danger"
                                  aria-label={t("common.deleteAria", { name: v.name })}
                                >
                                  <Icon name="trash" size={15} />
                                </button>
                              </div>
                            </td>
                          </tr>
                        );
                      }}
                    </For>
                  </tbody>
                </table>
              </div>
            </Show>
          </Show>
        </div>

        <Dialog
          open={createOpen()}
          onClose={() => setCreateOpen(false)}
          title={t("env.newVariable")}
          footer={
            <>
              <Button variant="secondary" onClick={() => setCreateOpen(false)}>
                {t("common.cancel")}
              </Button>
              <Button type="submit" form="new-variable-form" loading={busy()}>
                {t("env.createVariable")}
              </Button>
            </>
          }
        >
          <form id="new-variable-form" onSubmit={submitCreate} class="space-y-4">
            <Field label={t("common.name")}>
              <Input
                value={name()}
                onInput={(e) => setName(e.currentTarget.value)}
                placeholder={t("env.varNamePlaceholder")}
              />
            </Field>
            <Field label={t("env.value")} hint={t("env.encryptedHint")}>
              <Textarea
                rows={3}
                value={value()}
                onInput={(e) => setValue(e.currentTarget.value)}
                placeholder={t("env.varValuePlaceholder")}
              />
              <ValueGenerator onGenerate={(v) => setValue(v)} />
            </Field>
            <Show when={error()}>
              <p class="text-sm text-danger">{error()}</p>
            </Show>
          </form>
        </Dialog>

        <Dialog
          open={!!deleting()}
          onClose={() => setDeleting(null)}
          title={t("env.deleteVariable")}
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
            {t("env.deleteVariableConfirm", { name: deleting()?.name ?? "" })}
          </p>
          <Show when={error()}>
            <p class="mt-3 text-sm text-danger">{error()}</p>
          </Show>
        </Dialog>

        <Dialog
          open={importOpen()}
          onClose={closeImport}
          title={t("env.importTitle")}
          footer={
            <Button variant="secondary" onClick={closeImport}>
              {t("common.cancel")}
            </Button>
          }
        >
          <div class="space-y-3">
            <p class="text-sm text-ink-muted">{t("env.importHint")}</p>
            <div
              onClick={() => fileInput?.click()}
              onDragOver={onDragOver}
              onDragLeave={onDragLeave}
              onDrop={onDrop}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  fileInput?.click();
                }
              }}
              role="button"
              tabIndex={0}
              class={[
                "flex cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border border-dashed px-6 py-10 text-center transition-colors",
                dragging()
                  ? "border-brand bg-brand-subtle"
                  : "border-line bg-surface-2 hover:border-ink-faint",
              ].join(" ")}
            >
              <Icon name="upload" size={28} class={dragging() ? "text-brand" : "text-ink-faint"} />
              <p class="text-sm font-medium text-ink">{t("env.dropTitle")}</p>
              <p class="text-xs text-ink-muted">{t("env.dropBrowse")}</p>
              <input
                ref={(el) => (fileInput = el)}
                type="file"
                accept=".env,.json,.yaml,.yml"
                onChange={onImportFile}
                class="hidden"
              />
            </div>
            <Show when={importBusy()}>
              <p class="text-sm text-ink-muted">{t("common.loading")}</p>
            </Show>
            <Show when={importError()}>
              <p class="text-sm text-danger">{importError()}</p>
            </Show>
          </div>
        </Dialog>

        <Dialog
          open={exportOpen()}
          onClose={() => setExportOpen(false)}
          title={t("env.exportTitle")}
        >
          <div class="space-y-3">
            <p class="text-sm text-ink-muted">{t("env.exportHint")}</p>
            <div class="space-y-2">
              <For each={EXPORT_FORMATS}>
                {(format) => (
                  <button
                    type="button"
                    disabled={exportBusy()}
                    onClick={() => doExport(format)}
                    class="flex w-full items-center gap-3 rounded-lg border border-line bg-surface-2 px-3 py-2.5 text-left transition-colors hover:border-brand hover:bg-surface disabled:pointer-events-none disabled:opacity-60"
                  >
                    <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-brand-subtle text-brand">
                      <Icon name="code-bracket" size={15} />
                    </div>
                    <div class="min-w-0 flex-1">
                      <p class="text-sm font-medium text-ink">{format.label}</p>
                      <p class="text-xs text-ink-muted">.{format.ext}</p>
                    </div>
                    <Icon name="chevron-right" size={15} class="text-ink-faint" />
                  </button>
                )}
              </For>
            </div>
            <Show when={error()}>
              <p class="text-sm text-danger">{error()}</p>
            </Show>
          </div>
        </Dialog>
      </Show>
    </Show>
  );
}

// Show just the start and end of a long value, keeping the table on one line.
function truncateMiddle(value: string, head = 16, tail = 6): string {
  if (value.length <= head + tail + 1) return value;
  return `${value.slice(0, head)}…${value.slice(-tail)}`;
}

// --- Export helpers ---

interface FormatOption {
  id: "json" | "yaml" | "env";
  ext: string;
  label: string;
}

const EXPORT_FORMATS: FormatOption[] = [
  { id: "json", ext: "json", label: "JSON" },
  { id: "env", ext: "env", label: ".env" },
  { id: "yaml", ext: "yaml", label: "YAML" },
];

function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

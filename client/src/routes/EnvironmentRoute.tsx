// Variables table for an environment: tag filter, value reveal/copy, and file import.
import { createEffect, createMemo, createSignal, For, Show } from "solid-js";
import { A, useParams } from "@solidjs/router";
import {
  addVariable,
  db,
  importVariables,
  removeVariable,
  renameEnvironment,
  setEnvironmentDescription,
} from "../store";
import { values } from "../store/mock";
import { t } from "../lib/i18n";
import { createRevealController } from "../lib/reveal";
import { Badge, Button, Icon, Input, Textarea } from "../components/ui";
import { Dialog, EmptyState, Field, RevealCountdown, ValueGenerator } from "../components/shared";

export function EnvironmentRoute() {
  const params = useParams();
  const projectId = () => params.projectId ?? "";
  const envId = () => params.envId ?? "";

  const environment = createMemo(() => db.environments.find((e) => e.id === envId()));
  const variables = createMemo(() => db.variables.filter((v) => v.environmentId === envId()));
  const tags = createMemo(() => db.tags.filter((t) => t.projectId === projectId()));

  // Tag filter (matches at least one selected tag).
  const [selectedTags, setSelectedTags] = createSignal<string[]>([]);
  const toggleTag = (id: string) =>
    setSelectedTags((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));

  const filtered = createMemo(() =>
    selectedTags().length === 0
      ? variables()
      : variables().filter((v) => v.tagIds.some((id) => selectedTags().includes(id))),
  );

  // Value reveal + copy. Revealed values auto-hide after 20s, with a countdown.
  const { isRevealed, countdownFor, toggle: toggleReveal } = createRevealController();
  const [copiedId, setCopiedId] = createSignal<string | null>(null);

  const copy = async (id: string) => {
    try {
      await navigator.clipboard.writeText(values[id] ?? "");
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
  const submitCreate = (e: Event) => {
    e.preventDefault();
    const n = name().trim();
    if (!n) return;
    addVariable(envId(), n, value());
    setName("");
    setValue("");
    setCreateOpen(false);
  };

  // Delete variable.
  const [deleting, setDeleting] = createSignal<{
    id: string;
    name: string;
  } | null>(null);

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
  const commitRename = () => {
    const n = draftName().trim();
    if (n) renameEnvironment(envId(), n);
    setRenaming(false);
  };
  const onRenameKey = (e: KeyboardEvent) => {
    if (e.key === "Enter") commitRename();
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
  const commitDesc = () => {
    setEnvironmentDescription(envId(), draftDesc().trim());
    setEditingDesc(false);
  };
  const onDescKey = (e: KeyboardEvent) => {
    if (e.key === "Enter") commitDesc();
    else if (e.key === "Escape") setEditingDesc(false);
  };

  // Import.
  let fileInput: HTMLInputElement | undefined;
  const [importOpen, setImportOpen] = createSignal(false);
  const [importError, setImportError] = createSignal<string | null>(null);
  const [dragging, setDragging] = createSignal(false);

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
    try {
      const entries = parseVariablesFile(await file.text(), file.name);
      importVariables(envId(), entries);
      setImportError(null);
      setImportOpen(false);
    } catch {
      setImportError(t("env.importInvalid"));
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

  const entries = createMemo<[string, string][]>(() =>
    variables()
      .map((v) => [v.name, values[v.id] ?? ""] as [string, string])
      .sort((a, b) => a[0].localeCompare(b[0])),
  );

  const doExport = (format: FormatOption) => {
    const slug =
      (environment()?.name ?? "environment")
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-+|-+$/g, "") || "environment";
    downloadFile(`${slug}.${format.ext}`, format.serialize(entries()), format.mime);
    setExportOpen(false);
  };

  return (
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
          <Icon name="arrow-left" size={14} /> {db.projects.find((p) => p.id === projectId())?.name}
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

        <Show when={tags().length > 0}>
          <div class="mb-4 flex flex-wrap items-center gap-2">
            <span class="mr-1 text-xs font-medium uppercase tracking-wide text-ink-faint">
              {t("env.filter")}
            </span>
            <For each={tags()}>
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

        <Show
          when={filtered().length > 0}
          fallback={
            <EmptyState
              icon="lock"
              title={variables().length === 0 ? t("env.noVariables") : t("env.noVariablesMatch")}
              description={variables().length === 0 ? t("env.addFirst") : t("env.tryRemoving")}
              action={
                variables().length === 0 ? (
                  <Button icon="plus" onClick={() => setCreateOpen(true)}>
                    {t("env.newVariable")}
                  </Button>
                ) : undefined
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
                <For each={filtered()}>
                  {(v) => {
                    const revealed = () => isRevealed(v.id);
                    const remaining = () => countdownFor(v.id);
                    const currentValue = () => values[v.id] ?? "";
                    const variableTags = () =>
                      v.tagIds.map((id) => db.tags.find((t) => t.id === id));
                    const isActive = () =>
                      db.variableVersions.some((vv) => vv.variableId === v.id && vv.active);

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
                              isActive() ? "bg-success" : "bg-ink-faint",
                            ].join(" ")}
                            title={isActive() ? t("variable.active") : t("variable.inactive")}
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
                              {(t) => (t ? <Badge variant="brand">{t.name}</Badge> : null)}
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
                              aria-label={t("common.deleteAria", {
                                name: v.name,
                              })}
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
            <Button type="submit" form="new-variable-form">
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
            <Button
              variant="danger"
              onClick={() => {
                const d = deleting();
                if (d) {
                  removeVariable(d.id);
                  setDeleting(null);
                }
              }}
            >
              {t("common.delete")}
            </Button>
          </>
        }
      >
        <p class="text-sm text-ink-muted">
          {t("env.deleteVariableConfirm", { name: deleting()?.name ?? "" })}
        </p>
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
                  onClick={() => doExport(format)}
                  class="flex w-full items-center gap-3 rounded-lg border border-line bg-surface-2 px-3 py-2.5 text-left transition-colors hover:border-brand hover:bg-surface"
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
        </div>
      </Dialog>
    </Show>
  );
}

// Show just the start and end of a long value, keeping the table on one line.
function truncateMiddle(value: string, head = 16, tail = 6): string {
  if (value.length <= head + tail + 1) return value;
  return `${value.slice(0, head)}…${value.slice(-tail)}`;
}

// Parse a .env, .json, or .yaml file into a key/value object. Throws on invalid format.
function parseVariablesFile(content: string, filename: string): Record<string, string> {
  const ext = filename.split(".").pop()?.toLowerCase() ?? "";
  const out: Record<string, string> = {};

  if (ext === "json") {
    const data = JSON.parse(content);
    if (typeof data !== "object" || data === null || Array.isArray(data))
      throw new Error("invalid json");
    for (const [k, v] of Object.entries(data)) out[k] = String(v);
    return out;
  }

  const isYaml = ext === "yaml" || ext === "yml";
  for (const rawLine of content.split(/\r?\n/)) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;

    let key = "";
    let val = "";
    if (isYaml) {
      const m = /^([^:]+):\s*(.*)$/.exec(line);
      if (!m) continue;
      key = m[1].trim();
      val = m[2].trim();
    } else {
      const i = rawLine.indexOf("=");
      if (i <= 0) continue;
      key = rawLine.slice(0, i).trim();
      val = rawLine.slice(i + 1).trim();
    }

    if (!key) continue;
    if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
      val = val.slice(1, -1);
    }
    out[key] = val;
  }

  return out;
}

// --- Export helpers ---

interface FormatOption {
  ext: string;
  label: string;
  mime: string;
  serialize: (entries: [string, string][]) => string;
}

const EXPORT_FORMATS: FormatOption[] = [
  { ext: "json", label: "JSON", mime: "application/json", serialize: serializeJson },
  { ext: "env", label: ".env", mime: "text/plain", serialize: serializeEnv },
  { ext: "yaml", label: "YAML", mime: "text/yaml", serialize: serializeYaml },
];

function serializeJson(entries: [string, string][]): string {
  const obj: Record<string, string> = {};
  for (const [k, v] of entries) obj[k] = v;
  return `${JSON.stringify(obj, null, 2)}\n`;
}

function serializeEnv(entries: [string, string][]): string {
  return entries.map(([k, v]) => `${k}=${quoteEnv(v)}`).join("\n") + (entries.length ? "\n" : "");
}

function serializeYaml(entries: [string, string][]): string {
  return entries.map(([k, v]) => `${k}: ${quoteYaml(v)}`).join("\n") + (entries.length ? "\n" : "");
}

function quoteEnv(value: string): string {
  return /[\s#"=']/.test(value) ? JSON.stringify(value) : value;
}

function quoteYaml(value: string): string {
  return value === "" || /^[\s#]|[\s]$|:\s|#/.test(value) ? JSON.stringify(value) : value;
}

function downloadFile(filename: string, content: string, mime: string): void {
  const blob = new Blob([content], { type: mime });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

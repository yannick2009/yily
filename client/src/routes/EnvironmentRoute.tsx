// Environment page: header + rename/description editing, search/sort/tag filter, and the variables table.
// Create/delete/import/export are delegated to dialog components.
import { createEffect, createResource, createSignal, For, Show, onCleanup } from "solid-js";
import { A, useParams } from "@solidjs/router";
import { environments, projects, tags, variables } from "../api";
import { t } from "../lib/i18n";
import { errorMessage } from "../lib/errors";
import { createRevealController } from "../lib/reveal";
import { copyToClipboard } from "../lib/clipboard";
import { Button, Icon, Input } from "../components/ui";
import { EmptyState } from "../components/shared";
import {
  CreateVariableDialog,
  DeleteVariableDialog,
  ExportVariablesDialog,
  ImportVariablesDialog,
  SortPill,
  VariablesTable,
} from "../components/environment";

type SortField = "name" | "created_at";
type SortOrder = "asc" | "desc";

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

  // Server-side sort: two independent axes (column + direction). Clicking the
  // active pill toggles the direction; clicking the inactive one switches the
  // column while preserving the direction.
  const [sortField, setSortField] = createSignal<SortField>("name");
  const [sortOrder, setSortOrder] = createSignal<SortOrder>("asc");
  const onSortClick = (field: SortField) => {
    if (sortField() === field) setSortOrder((o) => (o === "asc" ? "desc" : "asc"));
    else setSortField(field);
  };

  const [envRes, { refetch: refetchEnv }] = createResource(() => environments.get(envId()));
  const [projectRes] = createResource(() => projects.get(projectId()));
  const [varsRes, { refetch: refetchVars }] = createResource(
    () => ({
      envId: envId(),
      tagIds: selectedTags(),
      search: debouncedQuery(),
      sort: sortField(),
      order: sortOrder(),
    }),
    ({ envId, tagIds, search, sort, order }) =>
      variables.list(envId, { tagIds, search, sort, order }),
  );
  const [tagsRes] = createResource(() => tags.list(projectId()));

  const environment = () => envRes();
  const variableList = () => varsRes() ?? [];
  const tagList = () => tagsRes() ?? [];
  const filtering = () => debouncedQuery().trim() !== "" || selectedTags().length > 0;
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
    await copyToClipboard(revealedValues()[id] ?? "");
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 1500);
  };

  // Dialog open state.
  const [createOpen, setCreateOpen] = createSignal(false);
  const [deleting, setDeleting] = createSignal<{
    id: string;
    name: string;
  } | null>(null);
  const [importOpen, setImportOpen] = createSignal(false);
  const [exportOpen, setExportOpen] = createSignal(false);

  // Rename environment (inline).
  let renameInput: HTMLInputElement | undefined;
  const [renaming, setRenaming] = createSignal(false);
  const [draftName, setDraftName] = createSignal("");
  const [renameError, setRenameError] = createSignal<string | null>(null);

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
        setRenameError(errorMessage(err));
      }
    }
    setRenaming(false);
  };
  const onRenameKey = (e: KeyboardEvent) => {
    if (e.key === "Enter") void commitRename();
    else if (e.key === "Escape") setRenaming(false);
  };

  // Edit description (inline).
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
        setRenameError(errorMessage(err));
      }
    }
    setEditingDesc(false);
  };
  const onDescKey = (e: KeyboardEvent) => {
    if (e.key === "Enter") void commitDesc();
    else if (e.key === "Escape") setEditingDesc(false);
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
                  maxLength={50}
                />
              </Show>
              <Show when={renameError()}>
                <p class="mt-2 text-sm text-danger">{renameError()}</p>
              </Show>
            </div>
            <div class="flex gap-2">
              <Button variant="secondary" icon="download" onClick={() => setImportOpen(true)}>
                {t("env.import")}
              </Button>
              <Button variant="secondary" icon="download" onClick={() => setExportOpen(true)}>
                {t("env.export")}
              </Button>
              <Button icon="plus" onClick={() => setCreateOpen(true)}>
                {t("env.newVariable")}
              </Button>
            </div>
          </div>

          <Show when={hasVariables()}>
            <div class="mb-4 space-y-3">
              <div class="flex flex-wrap items-center gap-3">
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
                <div class="flex items-center gap-2">
                  <span class="text-xs font-medium uppercase tracking-wide text-ink-faint">
                    {t("env.sortBy")}
                  </span>
                  <div class="flex items-center gap-2" role="group" aria-label={t("env.sortBy")}>
                    <SortPill
                      label={t("env.sortName")}
                      active={sortField() === "name"}
                      order={sortOrder()}
                      onClick={() => onSortClick("name")}
                    />
                    <SortPill
                      label={t("env.sortDate")}
                      active={sortField() === "created_at"}
                      order={sortOrder()}
                      onClick={() => onSortClick("created_at")}
                    />
                  </div>
                </div>
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
              <VariablesTable
                variables={variableList()}
                projectId={projectId()}
                envId={envId()}
                isRevealed={(id) => reveal.isRevealed(id)}
                getRevealedValue={(id) => revealedValues()[id] ?? ""}
                remainingFor={(id) => reveal.countdownFor(id)}
                copiedId={copiedId()}
                onToggleReveal={toggleReveal}
                onCopy={copy}
                onDelete={setDeleting}
              />
            </Show>
          </Show>
        </div>

        <CreateVariableDialog
          open={createOpen()}
          onClose={() => setCreateOpen(false)}
          envId={envId()}
          onCreated={() => refetchVars()}
        />
        <DeleteVariableDialog
          target={deleting()}
          onClose={() => setDeleting(null)}
          onDeleted={() => refetchVars()}
        />
        <ImportVariablesDialog
          open={importOpen()}
          onClose={() => setImportOpen(false)}
          envId={envId()}
          onImported={() => refetchVars()}
        />
        <ExportVariablesDialog
          open={exportOpen()}
          onClose={() => setExportOpen(false)}
          envId={envId()}
        />
      </Show>
    </Show>
  );
}

// Variable detail: current value, version history with activate/deactivate, and tag editing.
import { createEffect, createMemo, createResource, createSignal, For, Show } from "solid-js";
import { A, useParams } from "@solidjs/router";
import { environments, tags, variables } from "../api";
import { locale, t } from "../lib/i18n";
import { createRevealController } from "../lib/reveal";
import { normalizeVariableName } from "../lib/variable";
import { Badge, Button, Card, Icon, Input, Textarea } from "../components/ui";
import { Dialog, EmptyState, Field, RevealCountdown, ValueGenerator } from "../components/shared";

export function VariableDetailRoute() {
  const params = useParams();
  const projectId = () => params.projectId ?? "";
  const envId = () => params.envId ?? "";
  const variableId = () => params.variableId ?? "";

  const [variableRes, { refetch: refetchVariable }] = createResource(() =>
    variables.get(variableId()),
  );
  const [envRes] = createResource(() => environments.get(envId()));
  const [versionsRes, { refetch: refetchVersions }] = createResource(() =>
    variables.listVersions(variableId()),
  );
  const [projectTagsRes] = createResource(() => tags.list(projectId()));

  const variable = () => variableRes();
  const environment = () => envRes();

  // Newest first, mirroring GET /variables/:id/versions.
  const versions = createMemo(() => {
    const list = versionsRes() ?? [];
    return [...list].sort((a, b) => b.version - a.version);
  });
  const currentVersion = createMemo(() => versions().find((v) => v.active) ?? null);

  const variableTags = createMemo(() => variable()?.tags ?? []);
  const projectTags = () => projectTagsRes() ?? [];

  // Reveal the current value on demand (the server only exposes it this way).
  const reveal = createRevealController();
  const [revealedValue, setRevealedValue] = createSignal<string | null>(null);
  const [copied, setCopied] = createSignal(false);

  const toggleCurrent = async () => {
    if (reveal.isRevealed("current")) {
      reveal.hide("current");
      setRevealedValue(null);
      return;
    }
    try {
      const value = await variables.getValue(variableId());
      setRevealedValue(value);
      reveal.show("current");
    } catch {
      /* reveal failed; leave hidden */
    }
  };

  const copyCurrent = async () => {
    try {
      await navigator.clipboard.writeText(revealedValue() ?? "");
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard unavailable */
    }
  };

  // New version.
  const [newOpen, setNewOpen] = createSignal(false);
  const [value, setValue] = createSignal("");
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const submitNew = async (e: Event) => {
    e.preventDefault();
    const v = value().trim();
    if (!v) return;
    setBusy(true);
    setError(null);
    try {
      await variables.addVersion(variableId(), v);
      setValue("");
      setNewOpen(false);
      setRevealedValue(null);
      reveal.hide("current");
      await refetchVersions();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.save"));
    } finally {
      setBusy(false);
    }
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
  const commitRename = async () => {
    const n = normalizeVariableName(draft());
    if (n && variable()) {
      try {
        await variables.rename(variableId(), n);
        await refetchVariable();
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

  // Edit tags.
  const [tagsOpen, setTagsOpen] = createSignal(false);
  const [selectedTagIds, setSelectedTagIds] = createSignal<string[]>([]);
  const openTags = () => {
    setSelectedTagIds((variable()?.tags ?? []).map((tag) => tag.id));
    setTagsOpen(true);
  };
  const toggleTag = (id: string) =>
    setSelectedTagIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));
  const saveTags = async () => {
    setBusy(true);
    setError(null);
    try {
      await variables.setTags(variableId(), selectedTagIds());
      setTagsOpen(false);
      await refetchVariable();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.save"));
    } finally {
      setBusy(false);
    }
  };

  // Clear value.
  const [clearOpen, setClearOpen] = createSignal(false);
  const clearValue = async () => {
    setBusy(true);
    setError(null);
    try {
      await variables.clearValue(variableId());
      setClearOpen(false);
      setRevealedValue(null);
      reveal.hide("current");
      await refetchVersions();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.delete"));
    } finally {
      setBusy(false);
    }
  };

  const toggleVersionActive = async (versionId: string, active: boolean) => {
    setBusy(true);
    setError(null);
    try {
      await variables.setVersionActive(variableId(), versionId, active);
      await refetchVersions();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("common.save"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Show
      when={!variableRes.loading}
      fallback={<p class="py-8 text-center text-sm text-ink-muted">{t("common.loading")}</p>}
    >
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
                  onInput={(e) => setDraft(normalizeVariableName(e.currentTarget.value))}
                  onKeyDown={onRenameKey}
                  onBlur={commitRename}
                  class="max-w-xs font-mono"
                />
              </Show>
              <div class="mt-2 flex flex-wrap items-center gap-1.5">
                <For each={variableTags()}>
                  {(tag) => <Badge variant="brand">{tag.name}</Badge>}
                </For>
                <Button size="sm" variant="ghost" icon="tag" onClick={openTags}>
                  {t("variable.editTags")}
                </Button>
              </div>
              <p class="mt-2 text-xs text-ink-muted">
                {t("variable.created", { date: formatDate(variable()!.created_at) })}
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
              fallback={
                versionsRes.loading ? (
                  <p class="text-sm text-ink-muted">{t("common.loading")}</p>
                ) : (
                  <p class="text-sm text-ink-muted">{t("variable.noActiveValue")}</p>
                )
              }
            >
              <div class="flex items-start gap-2">
                <code
                  class="min-w-0 flex-1 truncate font-mono text-sm text-ink-muted"
                  title={reveal.isRevealed("current") ? (revealedValue() ?? undefined) : undefined}
                >
                  {reveal.isRevealed("current") ? revealedValue() : "••••••••••"}
                </code>
                <Show when={reveal.isRevealed("current") && reveal.countdownFor("current") > 0}>
                  <RevealCountdown seconds={reveal.countdownFor("current")} />
                </Show>
                <div class="flex shrink-0 items-center gap-1">
                  <button
                    type="button"
                    onClick={toggleCurrent}
                    class="rounded-md p-1.5 text-ink-muted hover:bg-surface-2 hover:text-ink"
                    aria-label={
                      reveal.isRevealed("current")
                        ? t("variable.hideAria", { n: currentVersion()!.version })
                        : t("variable.revealAria", { n: currentVersion()!.version })
                    }
                  >
                    <Icon name={reveal.isRevealed("current") ? "eye-off" : "eye"} size={15} />
                  </button>
                  <button
                    type="button"
                    onClick={copyCurrent}
                    class="rounded-md p-1.5 text-ink-muted hover:bg-surface-2 hover:text-ink"
                    aria-label={t("variable.copyAria", { n: currentVersion()!.version })}
                  >
                    <Icon name={copied() ? "check" : "copy"} size={15} />
                  </button>
                </div>
              </div>
            </Show>
          </section>

          <section>
            <h2 class="mb-3 text-base font-semibold">{t("variable.versions")}</h2>
            <Show
              when={!versionsRes.loading}
              fallback={
                <p class="py-8 text-center text-sm text-ink-muted">{t("common.loading")}</p>
              }
            >
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
                    {(vv) => (
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
                            {t("variable.created", { date: formatDate(vv.created_at) })}
                          </p>
                        </div>
                        <div class="flex shrink-0 items-center gap-1">
                          <Button
                            size="sm"
                            variant={vv.active ? "secondary" : "primary"}
                            loading={busy()}
                            onClick={() => toggleVersionActive(vv.id, !vv.active)}
                          >
                            {vv.active ? t("variable.deactivate") : t("variable.activate")}
                          </Button>
                        </div>
                      </div>
                    )}
                  </For>
                </Card>
              </Show>
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
              <Button type="submit" form="new-version-form" loading={busy()}>
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
            <Show when={error()}>
              <p class="text-sm text-danger">{error()}</p>
            </Show>
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
              <Button onClick={saveTags} loading={busy()}>
                {t("common.save")}
              </Button>
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
          <Show when={error()}>
            <p class="mt-3 text-sm text-danger">{error()}</p>
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
              <Button variant="danger" onClick={clearValue} loading={busy()}>
                {t("variable.clearValue")}
              </Button>
            </>
          }
        >
          <p class="text-sm text-ink-muted">
            {t("variable.clearValueConfirm", { name: variable()?.name ?? "" })}
          </p>
          <Show when={error()}>
            <p class="mt-3 text-sm text-danger">{error()}</p>
          </Show>
        </Dialog>
      </Show>
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

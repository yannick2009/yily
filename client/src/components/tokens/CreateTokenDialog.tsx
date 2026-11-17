// Create-token dialog. Shows a form, then the secret once on success (one-time reveal).
import { createResource, createSignal, For, Show } from "solid-js";
import { environments, projects, tokens } from "../../api";
import { t } from "../../lib/i18n";
import { errorMessage } from "../../lib/errors";
import { copyToClipboard } from "../../lib/clipboard";
import type { TokenScope } from "../../types";
import { Button, Icon, Input } from "../ui";
import { Dialog, Field } from "../shared";
import { ExpirationPicker } from "./ExpirationPicker";
import { ScopePicker } from "./ScopePicker";
import { DEFAULT_EXPIRATION_DAYS, DEFAULT_SCOPE, EXPIRATION_NEVER } from "./constants";

export function CreateTokenDialog(props: {
  open: boolean;
  onClose: () => void;
  onCreated: () => unknown;
}) {
  const [name, setName] = createSignal("");
  const [scope, setScope] = createSignal<TokenScope>(DEFAULT_SCOPE);
  const [days, setDays] = createSignal<number>(DEFAULT_EXPIRATION_DAYS);
  const [projectId, setProjectId] = createSignal("");
  const [environmentId, setEnvironmentId] = createSignal("");
  const [created, setCreated] = createSignal<string | null>(null);
  const [copied, setCopied] = createSignal(false);
  const [busy, setBusy] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  const [projectList] = createResource(
    () => (props.open ? true : false),
    () => projects.list(),
  );
  const [environmentList] = createResource(
    () => (props.open && projectId() !== "" ? projectId() : false),
    (pid) => environments.list(pid),
  );

  const reset = () => {
    setName("");
    setScope(DEFAULT_SCOPE);
    setDays(DEFAULT_EXPIRATION_DAYS);
    setProjectId("");
    setEnvironmentId("");
    setCreated(null);
    setCopied(false);
    setError(null);
  };

  const close = () => {
    reset();
    props.onClose();
  };

  // A write-scoped token must expire; switching to write resets a "never" selection to the default preset.
  const changeScope = (next: TokenScope) => {
    setScope(next);
    if (next === "write" && days() === EXPIRATION_NEVER) setDays(DEFAULT_EXPIRATION_DAYS);
  };

  const changeProject = (nextProjectId: string) => {
    setProjectId(nextProjectId);
    setEnvironmentId("");
  };

  const submit = async (e: Event) => {
    e.preventDefault();
    const n = name().trim();
    if (!n) return;
    setBusy(true);
    setError(null);
    try {
      const d = days();
      const createdToken = await tokens.create({
        name: n,
        scope: scope(),
        expires_in_days: d === EXPIRATION_NEVER ? null : d,
        project_id: projectId() || null,
        environment_id: projectId() && environmentId() ? environmentId() : null,
      });
      setCreated(createdToken.token);
      setName("");
      await props.onCreated();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const copy = async () => {
    const s = created();
    if (!s) return;
    if (await copyToClipboard(s)) setCopied(true);
  };

  return (
    <Dialog
      open={props.open}
      onClose={close}
      title={created() === null ? t("tokens.createTitle") : t("tokens.createdTitle")}
      footer={
        created() === null ? (
          <>
            <Button variant="secondary" onClick={close}>
              {t("common.cancel")}
            </Button>
            <Button type="submit" form="create-token-form" loading={busy()}>
              {t("tokens.create")}
            </Button>
          </>
        ) : (
          <Button variant="secondary" onClick={close}>
            {t("tokens.done")}
          </Button>
        )
      }
    >
      <Show
        when={created() === null}
        fallback={
          <div class="space-y-4">
            <div class="rounded-lg border border-line bg-surface-2 p-3">
              <p class="break-all font-mono text-sm text-ink">{created()}</p>
            </div>
            <div class="flex items-start gap-2 rounded-lg bg-brand-subtle p-3 text-sm text-brand">
              <Icon name="lock" size={16} class="mt-0.5 shrink-0" />
              <p>{t("tokens.copyNow")}</p>
            </div>
            <Button
              variant="secondary"
              icon={copied() ? "check" : "copy"}
              onClick={copy}
              class="w-full"
            >
              {copied() ? t("tokens.copied") : t("tokens.copy")}
            </Button>
          </div>
        }
      >
        <form id="create-token-form" onSubmit={submit} class="space-y-4">
          <Field label={t("common.name")} hint={t("tokens.nameHint")}>
            <Input
              value={name()}
              onInput={(e) => setName(e.currentTarget.value)}
              placeholder={t("tokens.namePlaceholder")}
            />
          </Field>
          <Field label={t("tokens.scopeLabel")} hint={t("tokens.scopeHint")}>
            <ScopePicker value={scope()} onChange={changeScope} />
          </Field>
          <Field label={t("tokens.projectLabel")} hint={t("tokens.projectHint")}>
            <select
              class="input"
              value={projectId()}
              onChange={(e) => changeProject(e.currentTarget.value)}
            >
              <option value="">{t("tokens.allProjects")}</option>
              <For each={projectList() ?? []}>
                {(project) => <option value={project.id}>{project.name}</option>}
              </For>
            </select>
          </Field>
          <Show when={projectId() !== ""}>
            <Field label={t("tokens.environmentLabel")} hint={t("tokens.environmentHint")}>
              <select
                class="input"
                value={environmentId()}
                onChange={(e) => setEnvironmentId(e.currentTarget.value)}
              >
                <option value="">{t("tokens.allEnvironments")}</option>
                <For each={environmentList() ?? []}>
                  {(env) => <option value={env.id}>{env.name}</option>}
                </For>
              </select>
            </Field>
          </Show>
          <Field label={t("tokens.expirationLabel")} hint={t("tokens.expirationHint")}>
            <ExpirationPicker value={days()} onChange={setDays} allowNever={scope() === "read"} />
          </Field>
          <Show when={error()}>
            <p class="text-sm text-danger">{error()}</p>
          </Show>
        </form>
      </Show>
    </Dialog>
  );
}

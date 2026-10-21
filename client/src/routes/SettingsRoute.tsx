// Settings: language picker, about section, and the danger zone (delete everything).
import { createSignal, For } from "solid-js";
import { Button, Icon } from "../components/ui";
import { Dialog } from "../components/shared";
import { LANGUAGES, applyLang, lang, t, type LangCode } from "../lib/i18n";
import { clearAllProjects, db } from "../store";

export function SettingsRoute() {
  const [clearAllOpen, setClearAllOpen] = createSignal(false);

  return (
    <div class="mx-auto max-w-xl space-y-6">
      <h1 class="text-lg font-semibold text-ink">{t("settings.title")}</h1>

      <section class="rounded-xl border border-line bg-surface p-4">
        <h2 class="text-sm font-semibold text-ink">{t("settings.language")}</h2>
        <select
          class="input mt-3"
          value={lang()}
          onChange={(e) => applyLang((e.currentTarget as HTMLSelectElement).value as LangCode)}
        >
          <For each={LANGUAGES}>{(l) => <option value={l.code}>{l.native}</option>}</For>
        </select>
      </section>

      <section class="rounded-xl border border-line bg-surface p-4">
        <h2 class="text-sm font-semibold text-ink">{t("settings.about")}</h2>

        <div class="mt-4 flex flex-col items-center text-center">
          <img src="/logo.svg" alt="Yily" class="h-16 w-auto" />
          <p class="mt-2 text-sm text-ink-muted">{t("about.tagline")}</p>
          <p class="mt-1 text-xs text-ink-faint">v0.1.0 · {t("about.licenseValue")}</p>

          <div class="mt-4 flex items-center gap-1">
            <a
              href="https://github.com/yannick2009/yily"
              target="_blank"
              rel="noreferrer"
              class="rounded-md p-2 text-ink-muted hover:bg-surface-2 hover:text-ink"
              title="GitHub"
              aria-label="GitHub"
            >
              <Icon name="github" size={18} />
            </a>
            <a
              href="https://yily.io"
              target="_blank"
              rel="noreferrer"
              class="rounded-md p-2 text-ink-muted hover:bg-surface-2 hover:text-ink"
              title={t("common.website")}
              aria-label={t("common.website")}
            >
              <Icon name="globe-alt" size={18} />
            </a>
          </div>
        </div>
      </section>

      <section class="rounded-xl border border-line bg-surface p-4">
        <h2 class="text-sm font-semibold text-danger">{t("settings.dangerZone")}</h2>
        <p class="mt-1 text-xs text-ink-muted">{t("settings.dangerZoneHint")}</p>
        <div class="mt-3">
          <button
            type="button"
            disabled={db.projects.length === 0}
            onClick={() => setClearAllOpen(true)}
            class="inline-flex h-9 select-none items-center justify-center gap-1.5 whitespace-nowrap rounded-lg border border-line px-3.5 text-sm font-medium text-ink-muted transition-colors hover:border-danger-subtle hover:bg-danger-subtle hover:text-danger disabled:pointer-events-none disabled:opacity-60"
          >
            <Icon name="trash" size={15} />
            {t("projects.deleteAll")}
          </button>
        </div>
      </section>

      <Dialog
        open={clearAllOpen()}
        onClose={() => setClearAllOpen(false)}
        title={t("projects.deleteAllTitle")}
        footer={
          <>
            <Button variant="secondary" onClick={() => setClearAllOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="danger"
              onClick={() => {
                clearAllProjects();
                setClearAllOpen(false);
              }}
            >
              {t("projects.deleteAll")}
            </Button>
          </>
        }
      >
        <p class="text-sm text-ink-muted">{t("projects.deleteAllConfirm")}</p>
      </Dialog>
    </div>
  );
}

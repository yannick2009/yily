// Controlled modal rendered in a Portal. Escape, overlay click, initial focus
// and body scroll-lock are handled automatically while open.
import { createEffect, onCleanup, Show, type JSX } from "solid-js";
import { Portal } from "solid-js/web";
import { t } from "../../lib/i18n";
import { Icon } from "../ui/Icon";

export function Dialog(props: {
  open: boolean;
  onClose: () => void;
  title: string;
  children: JSX.Element;
  footer?: JSX.Element;
}) {
  let panel: HTMLDivElement | undefined;

  createEffect(() => {
    if (!props.open) return;

    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") props.onClose();
    };

    document.addEventListener("keydown", onKey);
    document.body.style.overflow = "hidden";
    queueMicrotask(() => panel?.querySelector<HTMLElement>("input, textarea, select, button")?.focus());

    onCleanup(() => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = "";
    });
  });

  return (
    <Show when={props.open}>
      <Portal>
        <div class="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto p-4 sm:items-center">
          <div class="absolute inset-0 bg-black/40" onClick={props.onClose} aria-hidden="true" />
          <div
            ref={(el) => (panel = el)}
            role="dialog"
            aria-modal="true"
            aria-label={props.title}
            class="relative my-8 w-full max-w-md rounded-xl border border-line bg-surface p-5 shadow-2xl"
          >
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-lg font-semibold text-ink">{props.title}</h2>
              <button
                type="button"
                onClick={props.onClose}
                class="rounded-md p-1 text-ink-muted hover:bg-surface-2 hover:text-ink"
                aria-label={t("common.close")}
              >
                <Icon name="x" />
              </button>
            </div>
            {props.children}
            {props.footer ? <div class="mt-5 flex justify-end gap-2">{props.footer}</div> : null}
          </div>
        </div>
      </Portal>
    </Show>
  );
}

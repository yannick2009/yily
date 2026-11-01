// Reactive controller for temporarily revealed secrets: each id is revealed for
// a fixed number of seconds (with a per-id countdown) then re-hides on its own.
//
// This only manages the reveal *state* (hidden → shown → auto-hidden). The
// secret value itself is never stored here: callers fetch it from the API
// (`variables.getValue`) and call `show(id)` once it is available.
import { createSignal, onCleanup } from "solid-js";

export function createRevealController(seconds = 20) {
  const [revealedIds, setRevealedIds] = createSignal<string[]>([]);
  const [countdowns, setCountdowns] = createSignal<Record<string, number>>({});
  const timers = new Map<string, ReturnType<typeof setInterval>>();

  const clearTimer = (id: string) => {
    const timer = timers.get(id);
    if (timer) {
      clearInterval(timer);
      timers.delete(id);
    }
  };

  // Hides the secret for the given id, clearing its countdown and timer.
  const hide = (id: string) => {
    clearTimer(id);
    setCountdowns((prev) => {
      const next = { ...prev };
      delete next[id];
      return next;
    });
    setRevealedIds((prev) => prev.filter((x) => x !== id));
  };

  // Reveals the secret for the given id, starting a countdown and timer to auto-hide.
  const show = (id: string) => {
    setRevealedIds((prev) => (prev.includes(id) ? prev : [...prev, id]));
    setCountdowns((prev) => ({ ...prev, [id]: seconds }));
    let remaining = seconds;
    timers.set(
      id,
      setInterval(() => {
        remaining -= 1;
        if (remaining <= 0) hide(id);
        else setCountdowns((prev) => ({ ...prev, [id]: remaining }));
      }, 1000),
    );
  };

  const isRevealed = (id: string) => revealedIds().includes(id);
  const countdownFor = (id: string) => countdowns()[id] ?? 0;

  onCleanup(() => {
    for (const timer of timers.values()) clearInterval(timer);
    timers.clear();
  });

  return { isRevealed, countdownFor, show, hide };
}

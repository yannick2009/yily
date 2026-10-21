import type { JSX } from "solid-js";

// Field component is a reusable form field wrapper that can display a label, hint, error message, and any child elements (like input fields). It provides a consistent layout and styling for form fields in the application.
export function Field(props: {
  label?: string;
  hint?: string;
  error?: string;
  children: JSX.Element;
  class?: string;
}) {
  return (
    <div class={["space-y-1.5", props.class].filter(Boolean).join(" ")}>
      {props.label ? (
        <span class="block text-sm font-medium text-ink">{props.label}</span>
      ) : null}
      {props.children}
      {props.error ? (
        <p class="text-sm text-danger">{props.error}</p>
      ) : props.hint ? (
        <p class="text-sm text-ink-muted">{props.hint}</p>
      ) : null}
    </div>
  );
}

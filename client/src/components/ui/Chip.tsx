// Shared chip primitive: compact labels, tag chips, and toggle chips (sort/filter).
// Keeps one visual recipe (radius, border, spacing, tones) for every chip-like control.
import { splitProps, type ComponentProps } from "solid-js";

export type ChipVariant = "neutral" | "brand" | "success" | "warning" | "danger";
export type ChipSize = "sm" | "md";

const BASE = "inline-flex items-center gap-1 whitespace-nowrap font-medium";

const SIZES: Record<ChipSize, string> = {
  sm: "rounded-md border px-2 py-0.5 text-xs",
  md: "rounded-md border px-3 py-1.5 text-sm",
};

// Static chips use a softened border so dense tables stay calm; toggle chips below
// keep a full-strength brand border when active to signal selection.
const TONES: Record<ChipVariant, string> = {
  brand: "border-brand/40 bg-brand-subtle text-brand",
  success: "border-success/40 bg-success-subtle text-success",
  warning: "border-warning/40 bg-warning-subtle text-warning",
  danger: "border-danger/40 bg-danger-subtle text-danger",
  neutral: "border-line bg-surface-2 text-ink-muted",
};

export function chipClass(options: { variant?: ChipVariant; size?: ChipSize } = {}): string {
  return [BASE, SIZES[options.size ?? "sm"], TONES[options.variant ?? "neutral"]].join(" ");
}

export function Chip(props: ComponentProps<"span"> & { variant?: ChipVariant; size?: ChipSize }) {
  const [local, rest] = splitProps(props, ["variant", "size", "class"]);
  return <span {...rest} class={[chipClass(local), local.class].filter(Boolean).join(" ")} />;
}

export function ChipButton(
  props: ComponentProps<"button"> & { active?: boolean; size?: ChipSize },
) {
  const [local, rest] = splitProps(props, ["active", "size", "class"]);
  return (
    <button
      type="button"
      {...rest}
      aria-pressed={local.active ?? false}
      class={[
        BASE,
        SIZES[local.size ?? "md"],
        "transition-colors",
        local.active
          ? "border-brand bg-brand-subtle text-brand"
          : "border-line bg-surface text-ink-muted hover:border-ink-faint hover:text-ink",
        local.class,
      ]
        .filter(Boolean)
        .join(" ")}
    />
  );
}

// Small inline label with neutral/brand/success color variants.
import { splitProps, type ComponentProps } from "solid-js";

export function Badge(
  props: ComponentProps<"span"> & { variant?: "neutral" | "brand" | "success" },
) {
  const [local, rest] = splitProps(props, ["variant", "class"]);
  const colors =
    local.variant === "brand"
      ? "bg-brand-subtle text-brand"
      : local.variant === "success"
        ? "bg-success-subtle text-success"
        : "bg-surface-2 text-ink-muted";

  return (
    <span
      {...rest}
      class={[
        "inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs font-medium",
        colors,
        local.class,
      ]
        .filter(Boolean)
        .join(" ")}
    />
  );
}

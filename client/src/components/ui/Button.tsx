// Reusable button: variant/size variants, optional leading/trailing icon,
// and a loading state (spinner + disabled).
import { splitProps, type ComponentProps } from "solid-js";
import { Icon, type IconName } from "./Icon";
import { Spinner } from "./Spinner";

type Variant = "primary" | "secondary" | "ghost" | "danger";
type Size = "sm" | "md";

export interface ButtonProps {
  variant?: Variant;
  size?: Size;
  icon?: IconName;
  iconRight?: IconName;
  loading?: boolean;
}

const base =
  "inline-flex select-none items-center justify-center gap-1.5 whitespace-nowrap rounded-lg text-sm font-medium transition-colors disabled:pointer-events-none disabled:opacity-60";

const variants: Record<Variant, string> = {
  primary: "bg-brand text-on-brand hover:bg-brand-hover",
  secondary: "border border-line bg-surface text-ink hover:bg-surface-2",
  ghost: "border border-line text-ink-muted hover:bg-surface-2 hover:text-ink",
  danger: "bg-danger-subtle text-danger hover:brightness-95",
};

const sizes: Record<Size, string> = {
  sm: "h-8 px-2.5",
  md: "h-9 px-3.5",
};

export function Button(props: ButtonProps & ComponentProps<"button">) {
  const [local, rest] = splitProps(props, [
    "variant",
    "size",
    "icon",
    "iconRight",
    "loading",
    "children",
    "class",
  ]);

  return (
    <button
      {...rest}
      class={[base, variants[local.variant ?? "primary"], sizes[local.size ?? "md"], local.class]
        .filter(Boolean)
        .join(" ")}
      disabled={local.loading || !!rest.disabled}
    >
      {local.loading ? (
        <Spinner size={14} />
      ) : local.icon ? (
        <Icon name={local.icon} size={15} />
      ) : null}
      {local.children}
      {local.iconRight && !local.loading ? <Icon name={local.iconRight} size={15} /> : null}
    </button>
  );
}

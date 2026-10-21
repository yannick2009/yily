// Surface container with rounded corners and a border.
import { splitProps, type ComponentProps } from "solid-js";

export function Card(props: ComponentProps<"div">) {
  const [local, rest] = splitProps(props, ["class"]);
  return <div {...rest} class={["rounded-xl border border-line bg-surface", local.class].filter(Boolean).join(" ")} />;
}

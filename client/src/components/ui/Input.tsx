// Styled <input> and <textarea> (classes .input / .textarea).
import { splitProps, type ComponentProps } from "solid-js";

export function Input(props: ComponentProps<"input">) {
  const [local, rest] = splitProps(props, ["class"]);
  return <input {...rest} class={["input", local.class].filter(Boolean).join(" ")} />;
}

export function Textarea(props: ComponentProps<"textarea">) {
  const [local, rest] = splitProps(props, ["class"]);
  return <textarea {...rest} class={["textarea resize-none", local.class].filter(Boolean).join(" ")} />;
}

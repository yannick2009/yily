// One column of the sort control. Clicking the active pill toggles the direction
// (indicated by the arrow); clicking the inactive one switches the sort column
// while preserving the current direction.
import { Show } from "solid-js";
import { ChipButton } from "../ui";

export function SortPill(props: {
  label: string;
  active: boolean;
  order: "asc" | "desc";
  onClick: () => void;
}) {
  return (
    <ChipButton active={props.active} onClick={props.onClick}>
      {props.label}
      <Show when={props.active}>
        <span aria-hidden="true">{props.order === "asc" ? "↑" : "↓"}</span>
      </Show>
    </ChipButton>
  );
}

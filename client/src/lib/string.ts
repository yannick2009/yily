// Show just the start and end of a long value, keeping single-line layouts intact.
export function truncateMiddle(value: string, head = 16, tail = 6): string {
  if (value.length <= head + tail + 1) return value;
  return `${value.slice(0, head)}…${value.slice(-tail)}`;
}

// normalizeVariableName uppercases a name and collapses whitespace runs to a
// single underscore ("my var" → "MY_VAR").
export function normalizeVariableName(name: string): string {
  return name.trim().toUpperCase().replace(/\s+/g, "_");
}

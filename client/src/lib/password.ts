// Password complexity helpers shared by the setup form.
//
// The policy mirrors the server's crypto.IsStrongPassword: in addition to the
// 12–128 character length bounds, a strong password must contain at least one
// uppercase letter, one lowercase letter, one digit and one special
// (non-alphanumeric) character.
export function isStrongPassword(password: string): boolean {
  return (
    /[A-Z]/.test(password) &&
    /[a-z]/.test(password) &&
    /[0-9]/.test(password) &&
    /[^A-Za-z0-9]/.test(password)
  );
}

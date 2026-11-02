// Translate a thrown value to a user-facing message.
// The server returns a stable `code` (e.g. "AUTH_INVALID_CREDENTIALS") in its
// error envelope. We look up a translation keyed by that code; if no
// translation exists for the current language, we fall back to the server's
// English message, then to a generic "something went wrong".
import { ApiError } from "../api/client";
import { t } from "./i18n";

export function errorMessage(err: unknown): string {
  if (err instanceof ApiError && err.code) {
    const key = `errors.${err.code}`;
    const translated = t(key);
    if (translated !== key) return translated;

    return err.message;
  }
  if (err instanceof Error) return err.message;
  return t("errors.INTERNAL_ERROR");
}

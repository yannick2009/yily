import type { TokenScope } from "../../types";

// Fixed set of lifetimes offered by the API. Must match server AllowedPATTTLDays.
export const EXPIRATION_PRESETS = [7, 30, 60, 90] as const;
export const DEFAULT_EXPIRATION_DAYS = 30;
// Value used by the expiration picker to represent "never expires" (read-only tokens).
export const EXPIRATION_NEVER = 0;
// Threshold (in days) below which a non-revoked token surfaces an "expiring soon" warning badge.
export const EXPIRY_WARNING_DAYS = 7;
export const MS_PER_DAY = 24 * 60 * 60 * 1000;
export const DEFAULT_SCOPE: TokenScope = "read";

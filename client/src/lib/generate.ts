// Secure random-value generators for secrets. Every generator relies on
// crypto.getRandomValues, so the output is cryptographically strong.
const HEX = "0123456789abcdef";
const URL_SAFE = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";
const PASSWORD_CHARS =
  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()-_=+[]{};:,.<>?";

// Human-typed password (mixed case, digits, symbols).
export function generatePassword(length = 16): string {
  return randomChars(PASSWORD_CHARS, length);
}

// API key: optional prefix + lowercase hex secret.
export function generateApiKey(length = 32, prefix = ""): string {
  return prefix + randomChars(HEX, length);
}

// RFC 4122 v4 UUID.
export function generateUuid(): string {
  // Native API where available (needs a secure context: HTTPS or localhost).
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  // Fallback for non-secure contexts: RFC 4122 v4 from secure random bytes.
  const b = randomBytes(16);
  b[6] = (b[6] & 0x0f) | 0x40;
  b[8] = (b[8] & 0x3f) | 0x80;
  const hex = Array.from(b, (x) => x.toString(16).padStart(2, "0")).join("");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

// Lowercase hex string (2 × `bytes` characters; 32 bytes = AES-256 key).
export function generateHex(bytes = 32): string {
  return Array.from(randomBytes(bytes), (x) => x.toString(16).padStart(2, "0")).join("");
}

// Standard base64 (with padding), like `openssl rand -base64 32`.
export function generateBase64(bytes = 32): string {
  return btoa(String.fromCharCode(...randomBytes(bytes)));
}

// URL-safe string (A–Z a–z 0–9 - _), e.g. a JWT/session secret.
export function generateUrlSafe(length = 32): string {
  return randomChars(URL_SAFE, length);
}

// Pick `length` characters from `alphabet`, avoiding modulo bias.
function randomChars(alphabet: string, length: number): string {
  const max = Math.floor(256 / alphabet.length) * alphabet.length;
  let out = "";
  while (out.length < length) {
    for (const b of randomBytes(length - out.length)) {
      if (b < max) out += alphabet[b % alphabet.length];
      if (out.length === length) break;
    }
  }
  return out;
}

// Fill a byte array from the secure RNG.
function randomBytes(length: number): Uint8Array {
  const bytes = new Uint8Array(length);
  crypto.getRandomValues(bytes);
  return bytes;
}

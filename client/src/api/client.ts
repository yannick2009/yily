// Thin fetch wrapper around the Yily API.
//
// Every response uses the `{ status, message, data }` envelope. This module
// unwraps `data` on success and throws an ApiError carrying the server's
// message on failure. Requests send the session cookie automatically
// (`credentials: "include"`), so the client is authenticated via the
// HttpOnly `yily_session` cookie set by the server.

// BASE_URL points at the Vite dev proxy (see vite.config.ts), which forwards
// /api → http://localhost:16529 in development.
const BASE_URL = "/api/v1";

// ApiError is thrown for any non-2xx response.
class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

// Envelope is the shape of every JSON response from the server.
interface Envelope<T> {
  status: number;
  message: string;
  data: T;
}

// Parse the JSON envelope regardless of status code; a non-JSON body (e.g. a
// proxy 502) becomes null and falls back to a generic message.
async function parseEnvelope<T>(res: Response): Promise<Envelope<T> | null> {
  try {
    return (await res.json()) as Envelope<T>;
  } catch {
    return null;
  }
}

// Build request headers, adding a JSON content-type only when a body is set
// and the caller did not already choose one (multipart forms must set their own).
function buildHeaders(init: RequestInit): Record<string, string> {
  const headers: Record<string, string> = { ...(init.headers as Record<string, string> | undefined) };
  if (init.body && !headers["Content-Type"] && !(init.body instanceof FormData)) {
    headers["Content-Type"] = "application/json";
  }
  return headers;
}

// request performs a JSON request and returns the unwrapped `data` payload.
export async function request<T>(method: string, path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(BASE_URL + path, {
    ...init,
    method,
    credentials: "include",
    cache: "no-store",
    headers: buildHeaders(init),
  });

  const envelope = await parseEnvelope<T>(res);

  if (!res.ok) {
    throw new ApiError(res.status, envelope?.message ?? `Request failed (${res.status})`);
  }

  return envelope?.data as T;
}

// requestRaw performs a request whose response body is not the JSON envelope
// (used for the environment export, which returns a file blob).
export async function requestRaw(path: string, init: RequestInit = {}): Promise<Response> {
  const res = await fetch(BASE_URL + path, {
    ...init,
    credentials: "include",
    cache: "no-store",
  });

  if (!res.ok) {
    const envelope = await parseEnvelope<unknown>(res);
    throw new ApiError(res.status, envelope?.message ?? `Request failed (${res.status})`);
  }

  return res;
}

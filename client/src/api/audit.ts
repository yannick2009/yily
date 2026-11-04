// Audit log API.
import { request, requestRaw } from "./client";
import type { AuditList } from "../types";

export interface AuditQuery {
  search?: string;
  actorType?: string;
  action?: string;
  from?: string; // Inclusive lower bound as YYYY-MM-DD (UTC) or RFC3339.
  to?: string; // Inclusive-day upper bound as YYYY-MM-DD (UTC) or exclusive RFC3339.
  limit?: number;
  offset?: number;
}

// list returns a page of audit events, newest first.
export async function list(query: AuditQuery = {}): Promise<AuditList> {
  const params = new URLSearchParams();
  if (query.search) params.set("search", query.search);
  if (query.actorType) params.set("actor_type", query.actorType);
  if (query.action) params.set("action", query.action);
  if (query.from) params.set("from", query.from);
  if (query.to) params.set("to", query.to);
  if (query.limit != null) params.set("limit", String(query.limit));
  if (query.offset != null) params.set("offset", String(query.offset));

  const qs = params.toString();
  const raw = await request<AuditList>("GET", `/audit${qs ? `?${qs}` : ""}`);
  return { events: raw.events ?? [], total: raw.total ?? 0 };
}

// exportCSV downloads the audit events matching the given filters as a CSV
// file and returns the raw bytes plus the server-suggested filename.
export async function exportCSV(query: AuditQuery = {}): Promise<{ blob: Blob; filename: string }> {
  const params = new URLSearchParams();
  if (query.search) params.set("search", query.search);
  if (query.actorType) params.set("actor_type", query.actorType);
  if (query.action) params.set("action", query.action);
  if (query.from) params.set("from", query.from);
  if (query.to) params.set("to", query.to);

  const qs = params.toString();
  const res = await requestRaw(`/audit/export${qs ? `?${qs}` : ""}`);
  const blob = await res.blob();

  const disposition = res.headers.get("Content-Disposition") ?? "";
  const match = /filename="([^"]+)"/.exec(disposition);
  const filename = match?.[1] ?? "audit-log.csv";

  return { blob, filename };
}

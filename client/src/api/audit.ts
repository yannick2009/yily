// Audit log API.
import { request } from "./client";
import type { AuditList } from "../types";

export interface AuditQuery {
  search?: string;
  actorType?: string;
  action?: string;
  limit?: number;
  offset?: number;
}

// list returns a page of audit events, newest first.
export async function list(query: AuditQuery = {}): Promise<AuditList> {
  const params = new URLSearchParams();
  if (query.search) params.set("search", query.search);
  if (query.actorType) params.set("actor_type", query.actorType);
  if (query.action) params.set("action", query.action);
  if (query.limit != null) params.set("limit", String(query.limit));
  if (query.offset != null) params.set("offset", String(query.offset));

  const qs = params.toString();
  const raw = await request<AuditList>("GET", `/audit${qs ? `?${qs}` : ""}`);
  return { events: raw.events ?? [], total: raw.total ?? 0 };
}

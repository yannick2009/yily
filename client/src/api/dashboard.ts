// Dashboard API.
import { request } from "./client";
import type { Dashboard } from "../types";

// get returns the aggregated dashboard data.
export async function get(): Promise<Dashboard> {
  return request<Dashboard>("GET", "/dashboard");
}

import { request } from "./client";
import type { VersionInfo } from "../types";

// getVersion returns the running server's version.
export async function getVersion(): Promise<VersionInfo> {
  return request<VersionInfo>("GET", "/version");
}

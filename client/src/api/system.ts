import { request } from "./client";
import type { UpdateInfo, VersionInfo } from "../types";

// getVersion returns the running server's version.
export async function getVersion(): Promise<VersionInfo> {
  return request<VersionInfo>("GET", "/version");
}

// getUpdate returns the latest available release, as cached by the server.
export async function getUpdate(): Promise<UpdateInfo> {
  return request<UpdateInfo>("GET", "/version/update");
}

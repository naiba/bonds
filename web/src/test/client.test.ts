import { describe, it, expect } from "vitest";
import axios from "axios";
import type { InternalAxiosRequestConfig, AxiosHeaders } from "axios";

describe("API Client", () => {
  it("can create an axios instance with /api baseURL", () => {
    const instance = axios.create({
      baseURL: "/api",
      headers: { "Content-Type": "application/json" },
    });
    expect(instance.defaults.baseURL).toBe("/api");
    expect(instance.defaults.headers["Content-Type"]).toBe("application/json");
  });

  it("AxiosHeaders can set and get Authorization", () => {
    const config = {
      headers: new axios.AxiosHeaders(),
    } as InternalAxiosRequestConfig;

    const token = "test-jwt-token";
    (config.headers as AxiosHeaders).set("Authorization", `Bearer ${token}`);

    expect(config.headers.get("Authorization")).toBe("Bearer test-jwt-token");
  });

  it("AxiosHeaders returns undefined for missing header", () => {
    const config = {
      headers: new axios.AxiosHeaders(),
    } as InternalAxiosRequestConfig;

    expect(config.headers.get("Authorization")).toBeUndefined();
  });
});

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { AxiosHeaders } from "axios";
import type { InternalAxiosRequestConfig } from "axios";
import { httpClient } from "@/api";

async function runRequestInterceptors(config: InternalAxiosRequestConfig) {
  const handlers = (httpClient.instance.interceptors.request as unknown as {
    handlers: Array<{ fulfilled?: (c: InternalAxiosRequestConfig) => InternalAxiosRequestConfig | Promise<InternalAxiosRequestConfig> }>;
  }).handlers;
  let current = config;
  for (const handler of handlers) {
    if (handler?.fulfilled) {
      current = await handler.fulfilled(current);
    }
  }
  return current;
}

function requestWith(authorization?: string): InternalAxiosRequestConfig {
  const headers = new AxiosHeaders();
  if (authorization) headers.set("Authorization", authorization);
  return { method: "post", url: "/auth/oauth/link", headers } as InternalAxiosRequestConfig;
}

// The OAuth binding page signs in with the entered password and then binds the
// provider identity with that account's token. The interceptor used to replace
// it with the token of whichever account was still signed in in this browser,
// so the identity was bound to the wrong account.
describe("request credentials supplied by the caller", () => {
  beforeEach(() => {
    localStorage.setItem("token", "token-of-the-account-still-signed-in");
  });
  afterEach(() => {
    localStorage.removeItem("token");
  });

  it("are sent as given instead of the stored session's token", async () => {
    const config = await runRequestInterceptors(requestWith("Bearer token-of-the-account-just-entered"));

    expect(config.headers.get("Authorization")).toBe("Bearer token-of-the-account-just-entered");
  });

  it("do not stop ordinary requests from carrying the stored session's token", async () => {
    const config = await runRequestInterceptors(requestWith());

    expect(config.headers.get("Authorization")).toBe("Bearer token-of-the-account-still-signed-in");
  });
});

import { describe, it, expect, beforeEach } from "vitest";
import { normalizeLanguageCode } from "@/i18n";
import i18n from "@/i18n";
import { httpClient } from "@/api";
import type { InternalAxiosRequestConfig, AxiosHeaders } from "axios";

// Mirror what axios feeds the interceptor. The interceptors at runtime walk
// through the registered handlers sequentially — call them the same way.
async function runRequestInterceptors(config: InternalAxiosRequestConfig) {
  const handlers = (httpClient.instance.interceptors.request as unknown as {
    handlers: Array<{ fulfilled?: (c: InternalAxiosRequestConfig) => InternalAxiosRequestConfig | Promise<InternalAxiosRequestConfig> }>;
  }).handlers;
  let current = config;
  for (const h of handlers) {
    if (h?.fulfilled) {
      current = await h.fulfilled(current);
    }
  }
  return current;
}

function makeConfig(): InternalAxiosRequestConfig {
  return {
    headers: {
      set(key: string, value: string) {
        (this as Record<string, unknown>)[key] = value;
        return this;
      },
    } as unknown as AxiosHeaders,
  } as InternalAxiosRequestConfig;
}

describe("Accept-Language request interceptor", () => {
  beforeEach(async () => {
    await i18n.changeLanguage("en");
  });

  it("sends Accept-Language matching the current i18n locale", async () => {
    await i18n.changeLanguage("zh");
    const config = await runRequestInterceptors(makeConfig());
    expect((config.headers as Record<string, unknown>)["Accept-Language"]).toBe("zh");
  });

  it("normalizes regional tags like zh-CN to a supported code", async () => {
    await i18n.changeLanguage("zh-CN");
    const config = await runRequestInterceptors(makeConfig());
    expect((config.headers as Record<string, unknown>)["Accept-Language"]).toBe("zh");
  });

  it("sends German Accept-Language when the UI is German", async () => {
    await i18n.changeLanguage("de");
    const config = await runRequestInterceptors(makeConfig());
    expect((config.headers as Record<string, unknown>)["Accept-Language"]).toBe("de");
  });

  it("falls back to en when the language is unsupported", async () => {
    await i18n.changeLanguage("ja");
    const config = await runRequestInterceptors(makeConfig());
    expect((config.headers as Record<string, unknown>)["Accept-Language"]).toBe("en");
  });
});

describe("normalizeLanguageCode", () => {
  it("maps regional zh tags to zh", () => {
    expect(normalizeLanguageCode("zh-CN")).toBe("zh");
    expect(normalizeLanguageCode("zh-Hans")).toBe("zh");
    expect(normalizeLanguageCode("de-DE")).toBe("de");
  });
  it("maps pt-BR and pt-PT to their region-specific codes", () => {
    expect(normalizeLanguageCode("pt-BR")).toBe("pt-BR");
    expect(normalizeLanguageCode("pt-PT")).toBe("pt-PT");
    expect(normalizeLanguageCode("pt-br")).toBe("pt-BR");
  });
  it("maps bare pt to pt-PT via region fallback", () => {
    expect(normalizeLanguageCode("pt")).toBe("pt-PT");
  });
  it("returns en for unsupported languages", () => {
    expect(normalizeLanguageCode("ja")).toBe("en");
    expect(normalizeLanguageCode(undefined)).toBe("en");
  });
});

import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  normalizeDomains,
  parseDomainTokens,
  createFormState,
  buildConfigPayload,
  getConfigSignature,
  normalizeStatus,
  fetchJson,
  postWithoutResponse,
  defaultFormState,
  defaultStatus,
} from "../utils";
import type { ConfigResponse, StatusResponse } from "../types";

// ─── normalizeDomains ────────────────────────────────────────────────────────

describe("normalizeDomains", () => {
  it("trims whitespace from each domain", () => {
    expect(normalizeDomains(["  example.com  ", " foo.com "])).toEqual([
      "example.com",
      "foo.com",
    ]);
  });

  it("deduplicates case-insensitively", () => {
    expect(
      normalizeDomains(["Example.COM", "example.com", "EXAMPLE.COM"]),
    ).toEqual(["Example.COM"]);
  });

  it("preserves original casing of the first occurrence", () => {
    expect(normalizeDomains(["Foo.COM", "foo.com"])).toEqual(["Foo.COM"]);
  });

  it("drops empty and whitespace-only entries", () => {
    expect(normalizeDomains(["", "  ", "valid.com"])).toEqual(["valid.com"]);
  });

  it("returns empty array for empty input", () => {
    expect(normalizeDomains([])).toEqual([]);
  });

  it("preserves order of first occurrences", () => {
    expect(normalizeDomains(["b.com", "a.com", "b.com"])).toEqual([
      "b.com",
      "a.com",
    ]);
  });
});

// ─── parseDomainTokens ───────────────────────────────────────────────────────

describe("parseDomainTokens", () => {
  it("splits on commas", () => {
    expect(parseDomainTokens("a.com,b.com")).toEqual(["a.com", "b.com"]);
  });

  it("splits on newlines", () => {
    expect(parseDomainTokens("a.com\nb.com")).toEqual(["a.com", "b.com"]);
  });

  it("deduplicates after splitting", () => {
    expect(parseDomainTokens("a.com,a.com")).toEqual(["a.com"]);
  });

  it("returns empty array for blank input", () => {
    expect(parseDomainTokens("")).toEqual([]);
  });
});

// ─── createFormState ─────────────────────────────────────────────────────────

const baseConfig: ConfigResponse = {
  autoStart: true,
  autoUpdateGfwList: false,
  bypassDomains: ["bypass.com"],
  companyDomains: ["corp.com"],
  companyIface: "eth1",
  defaultIface: "eth0",
  extraGfwDomains: ["extra.com"],
  gfwIface: "tun0",
  gfwProxy: "http://127.0.0.1:7890",
  gfwlistUrl: "https://example.com/gfw.txt",
  httpProxyIface: "lo",
  port: 1080,
  verboseLog: false,
};

describe("createFormState", () => {
  it("maps all fields correctly", () => {
    const form = createFormState(baseConfig);
    expect(form.autoStart).toBe(true);
    expect(form.companyIface).toBe("eth1");
    expect(form.defaultIface).toBe("eth0");
    expect(form.gfwIface).toBe("tun0");
    expect(form.gfwlistUrl).toBe("https://example.com/gfw.txt");
    expect(form.httpProxyIface).toBe("lo");
    expect(form.verboseLog).toBe(false);
  });

  it("converts port number to string", () => {
    expect(createFormState(baseConfig).port).toBe("1080");
  });

  it("falls back to port '1080' when config port is 0", () => {
    expect(createFormState({ ...baseConfig, port: 0 }).port).toBe("1080");
  });

  it("normalizes domain arrays", () => {
    const form = createFormState({
      ...baseConfig,
      bypassDomains: [" dup.com ", "dup.com"],
    });
    expect(form.bypassDomains).toEqual(["dup.com"]);
  });
});

// ─── buildConfigPayload ──────────────────────────────────────────────────────

describe("buildConfigPayload", () => {
  const form = createFormState(baseConfig);

  it("converts port string to number", () => {
    expect(buildConfigPayload(form, "").port).toBe(1080);
  });

  it("falls back to 1080 when port is not a valid number", () => {
    expect(buildConfigPayload({ ...form, port: "abc" }, "").port).toBe(1080);
  });

  it("uses effectiveGfwProxy trimmed", () => {
    expect(buildConfigPayload(form, "  http://proxy  ").gfwProxy).toBe(
      "http://proxy",
    );
  });

  it("trims gfwlistUrl", () => {
    expect(
      buildConfigPayload({ ...form, gfwlistUrl: "  https://x.com  " }, "")
        .gfwlistUrl,
    ).toBe("https://x.com");
  });

  it("normalizes domain arrays", () => {
    const payload = buildConfigPayload(
      { ...form, companyDomains: [" dup.com ", "dup.com"] },
      "",
    );
    expect(payload.companyDomains).toEqual(["dup.com"]);
  });

  it("passes through interface fields unchanged", () => {
    const payload = buildConfigPayload(form, "");
    expect(payload.companyIface).toBe("eth1");
    expect(payload.defaultIface).toBe("eth0");
    expect(payload.gfwIface).toBe("tun0");
  });
});

// ─── getConfigSignature ──────────────────────────────────────────────────────

describe("getConfigSignature", () => {
  it("returns same signature for equivalent configs", () => {
    const a = getConfigSignature(baseConfig);
    const b = getConfigSignature({ ...baseConfig });
    expect(a).toBe(b);
  });

  it("returns different signature when a field changes", () => {
    const a = getConfigSignature(baseConfig);
    const b = getConfigSignature({ ...baseConfig, port: 9090 });
    expect(a).not.toBe(b);
  });

  it("produces different signatures for same domain with different casing", () => {
    const a = getConfigSignature({
      ...baseConfig,
      bypassDomains: ["example.com"],
    });
    const b = getConfigSignature({
      ...baseConfig,
      bypassDomains: ["EXAMPLE.COM"],
    });
    // normalizeDomains deduplicates case-insensitively but preserves original casing
    expect(a).not.toBe(b);
  });

  it("trims gfwProxy before hashing", () => {
    const a = getConfigSignature({ ...baseConfig, gfwProxy: "http://p" });
    const b = getConfigSignature({ ...baseConfig, gfwProxy: "  http://p  " });
    expect(a).toBe(b);
  });
});

// ─── normalizeStatus ─────────────────────────────────────────────────────────

describe("normalizeStatus", () => {
  it("preserves all present fields", () => {
    const input: StatusResponse = {
      httpProxyIface: "eth0",
      httpProxyPort: 8080,
      logs: ["a"],
      port: 1080,
      running: true,
    };
    expect(normalizeStatus(input)).toEqual(input);
  });

  it("defaults logs to empty array when missing", () => {
    expect(normalizeStatus({ running: false }).logs).toEqual([]);
  });

  it("defaults port to 1080 when missing", () => {
    expect(normalizeStatus({ running: false }).port).toBe(1080);
  });
});

// ─── defaultFormState / defaultStatus ────────────────────────────────────────

describe("defaultFormState", () => {
  it("has port '1080'", () => {
    expect(defaultFormState.port).toBe("1080");
  });

  it("has empty domain arrays", () => {
    expect(defaultFormState.bypassDomains).toEqual([]);
    expect(defaultFormState.companyDomains).toEqual([]);
    expect(defaultFormState.extraGfwDomains).toEqual([]);
  });
});

describe("defaultStatus", () => {
  it("is not running", () => {
    expect(defaultStatus.running).toBe(false);
  });

  it("has empty logs", () => {
    expect(defaultStatus.logs).toEqual([]);
  });
});

// ─── fetchJson ───────────────────────────────────────────────────────────────

describe("fetchJson", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("returns parsed JSON on success", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ value: 42 }),
      }),
    );
    const result = await fetchJson<{ value: number }>("/api/test");
    expect(result).toEqual({ value: 42 });
  });

  it("sets Content-Type: application/json header", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    });
    vi.stubGlobal("fetch", mockFetch);
    await fetchJson("/api/test");
    const headers: Headers = mockFetch.mock.calls[0][1].headers;
    expect(headers.get("Content-Type")).toBe("application/json");
  });

  it("does not override an existing Content-Type header", async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    });
    vi.stubGlobal("fetch", mockFetch);
    await fetchJson("/api/test", {
      headers: { "Content-Type": "text/plain" },
    });
    const headers: Headers = mockFetch.mock.calls[0][1].headers;
    expect(headers.get("Content-Type")).toBe("text/plain");
  });

  it("throws with server message on non-ok response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        text: () => Promise.resolve("Bad request"),
      }),
    );
    await expect(fetchJson("/api/test")).rejects.toThrow("Bad request");
  });

  it("throws with status code when server sends empty error body", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        text: () => Promise.resolve(""),
      }),
    );
    await expect(fetchJson("/api/test")).rejects.toThrow("500");
  });
});

// ─── postWithoutResponse ──────────────────────────────────────────────────────

describe("postWithoutResponse", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("sends POST with JSON-serialized body", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true });
    vi.stubGlobal("fetch", mockFetch);
    await postWithoutResponse("/api/action", { key: "val" });
    expect(mockFetch).toHaveBeenCalledWith(
      "/api/action",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ key: "val" }),
      }),
    );
  });

  it("sends POST with no body when body is undefined", async () => {
    const mockFetch = vi.fn().mockResolvedValue({ ok: true });
    vi.stubGlobal("fetch", mockFetch);
    await postWithoutResponse("/api/action");
    const init = mockFetch.mock.calls[0][1];
    expect(init.body).toBeUndefined();
  });

  it("throws on non-ok response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 403,
        text: () => Promise.resolve("Forbidden"),
      }),
    );
    await expect(postWithoutResponse("/api/action")).rejects.toThrow(
      "Forbidden",
    );
  });
});

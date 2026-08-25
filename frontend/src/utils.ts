import type { ConfigResponse, FormState, StatusResponse } from "./types";

export const defaultFormState: FormState = {
  autoStart: false,
  autoUpdateGfwList: false,
  bypassDomains: [],
  companyDomains: [],
  companyIface: "",
  defaultIface: "",
  extraGfwDomains: [],
  gfwIface: "",
  gfwlistUrl: "",
  httpProxyIface: "",
  port: "1080",
  verboseLog: false,
};

export const defaultStatus: StatusResponse = {
  logs: [],
  port: 1080,
  running: false,
};

export function normalizeDomains(domains: string[]): string[] {
  const seen = new Set<string>();

  return domains.reduce<string[]>((items, domain) => {
    const trimmedDomain = domain.trim();
    const normalizedDomain = trimmedDomain.toLowerCase();

    if (!trimmedDomain || seen.has(normalizedDomain)) {
      return items;
    }

    seen.add(normalizedDomain);
    items.push(trimmedDomain);
    return items;
  }, []);
}

export function parseDomainTokens(value: string): string[] {
  return normalizeDomains(value.split(/[,\n]/));
}

export function createFormState(config: ConfigResponse): FormState {
  return {
    autoStart: config.autoStart ?? false,
    autoUpdateGfwList: config.autoUpdateGfwList ?? false,
    bypassDomains: normalizeDomains(config.bypassDomains ?? []),
    companyDomains: normalizeDomains(config.companyDomains ?? []),
    companyIface: config.companyIface ?? "",
    defaultIface: config.defaultIface ?? "",
    extraGfwDomains: normalizeDomains(config.extraGfwDomains ?? []),
    gfwIface: config.gfwIface ?? "",
    gfwlistUrl: config.gfwlistUrl ?? "",
    httpProxyIface: config.httpProxyIface ?? "",
    port: String(config.port || 1080),
    verboseLog: config.verboseLog ?? false,
  };
}

export function buildConfigPayload(
  form: FormState,
  effectiveGfwProxy: string,
): ConfigResponse {
  return {
    autoStart: form.autoStart,
    autoUpdateGfwList: form.autoUpdateGfwList,
    bypassDomains: normalizeDomains(form.bypassDomains),
    companyDomains: normalizeDomains(form.companyDomains),
    companyIface: form.companyIface,
    defaultIface: form.defaultIface,
    extraGfwDomains: normalizeDomains(form.extraGfwDomains),
    gfwIface: form.gfwIface,
    gfwProxy: effectiveGfwProxy.trim(),
    gfwlistUrl: form.gfwlistUrl.trim(),
    httpProxyIface: form.httpProxyIface,
    port: Number.parseInt(form.port, 10) || 1080,
    verboseLog: form.verboseLog,
  };
}

export function getConfigSignature(config: ConfigResponse): string {
  return JSON.stringify({
    autoStart: config.autoStart,
    autoUpdateGfwList: config.autoUpdateGfwList,
    bypassDomains: normalizeDomains(config.bypassDomains),
    companyDomains: normalizeDomains(config.companyDomains),
    companyIface: config.companyIface,
    defaultIface: config.defaultIface,
    extraGfwDomains: normalizeDomains(config.extraGfwDomains),
    gfwIface: config.gfwIface,
    gfwProxy: config.gfwProxy.trim(),
    gfwlistUrl: config.gfwlistUrl.trim(),
    httpProxyIface: config.httpProxyIface,
    port: config.port,
    verboseLog: config.verboseLog,
  });
}

export function normalizeStatus(nextStatus: StatusResponse): StatusResponse {
  return {
    httpProxyIface: nextStatus.httpProxyIface,
    httpProxyPort: nextStatus.httpProxyPort,
    logs: nextStatus.logs ?? [],
    port: nextStatus.port ?? 1080,
    running: nextStatus.running,
  };
}

async function assertResponseOk(response: Response): Promise<void> {
  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }
}

export async function fetchJson<T>(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<T> {
  const headers = new Headers(init?.headers);

  if (!headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(input, {
    ...init,
    headers,
  });

  await assertResponseOk(response);
  return (await response.json()) as T;
}

export async function postWithoutResponse(url: string, body?: unknown) {
  const response = await fetch(url, {
    body: body === undefined ? undefined : JSON.stringify(body),
    headers: {
      "Content-Type": "application/json",
    },
    method: "POST",
  });

  await assertResponseOk(response);
}

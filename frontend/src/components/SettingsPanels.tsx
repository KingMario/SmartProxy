import type {
  DetectResponse,
  DetectTarget,
  FormState,
  NetworkInterface,
  ToastState,
} from "../types";
import { fetchJson } from "../utils";
import { InterfaceSelector } from "./InterfaceSelector";
import TokenInput from "./TokenInput";

type FieldSetter = <K extends keyof FormState>(
  key: K,
  value: FormState[K],
) => void;

type QuickSettingsPanelProps = {
  autoStart: boolean;
  autoUpdateGFWList: boolean;
  verboseLog: boolean;
  detectedSystemProxy: string;
  isRefreshingSystemProxy: boolean;
  onAutoStartChange: (value: boolean) => void;
  onAutoUpdateGFWListChange: (value: boolean) => void;
  onVerboseLogChange: (value: boolean) => void;
  onRefreshSystemProxy: () => void;
  setUseDetectedSystemProxy: (value: boolean) => void;
  useDetectedSystemProxy: boolean;
};

export function QuickSettingsPanel({
  autoStart,
  autoUpdateGFWList,
  verboseLog,
  detectedSystemProxy,
  isRefreshingSystemProxy,
  onAutoStartChange,
  onAutoUpdateGFWListChange,
  onVerboseLogChange,
  onRefreshSystemProxy,
  setUseDetectedSystemProxy,
  useDetectedSystemProxy,
}: QuickSettingsPanelProps) {
  return (
    <section className="panel">
      <header className="panel__header">Quick Settings</header>
      <div className="panel__body">
        <label className="switch" htmlFor="auto-start-toggle">
          <input
            checked={autoStart}
            id="auto-start-toggle"
            onChange={(event) => {
              onAutoStartChange(event.target.checked);
            }}
            type="checkbox"
          />
          <span>Auto-start proxy on program launch</span>
        </label>
        <label className="switch" htmlFor="auto-update-gfwlist-toggle">
          <input
            checked={autoUpdateGFWList}
            id="auto-update-gfwlist-toggle"
            onChange={(event) => {
              onAutoUpdateGFWListChange(event.target.checked);
            }}
            type="checkbox"
          />
          <span>Auto-update GFWList on program launch</span>
        </label>
        <label className="switch" htmlFor="verbose-log-toggle">
          <input
            checked={verboseLog}
            id="verbose-log-toggle"
            onChange={(event) => {
              onVerboseLogChange(event.target.checked);
            }}
            type="checkbox"
          />
          <span>Verbose Logging (Show routing decisions)</span>
        </label>
        <div className="field field--compact">
          <label
            className="proxy-toggle"
            htmlFor="global-proxy-toggle"
            id="global-proxy-toggle-label"
          >
            <input
              checked={useDetectedSystemProxy && Boolean(detectedSystemProxy)}
              disabled={!detectedSystemProxy || isRefreshingSystemProxy}
              id="global-proxy-toggle"
              onChange={(event) => {
                setUseDetectedSystemProxy(event.target.checked);
              }}
              type="checkbox"
            />
            <span className="proxy-toggle__content">
              {detectedSystemProxy
                ? `Global Proxy Detected (${detectedSystemProxy}). Use it as GFW Upstream Proxy?`
                : "Global Proxy Not Detected in system HTTP proxy settings."}{" "}
              <button
                className="button button--link"
                disabled={isRefreshingSystemProxy}
                onClick={onRefreshSystemProxy}
                type="button"
              >
                {isRefreshingSystemProxy ? "Refreshing..." : "Refresh"}
              </button>
            </span>
          </label>
          <p className="field-description" id="global-proxy-source-description">
            Source: Smart Proxy reads the OS-level HTTP proxy setting, not the
            HTTP Proxy panel below.
          </p>
        </div>
      </div>
    </section>
  );
}

type GeneralSettingsPanelProps = {
  form: FormState;
  gfwProxyActive: boolean;
  interfaceOptions: NetworkInterface[];
  isLoading: boolean;
  onInterfacesLoaded: (interfaces: NetworkInterface[]) => void;
  setField: FieldSetter;
  setToast: (toast: ToastState) => void;
};

function makeDetectHandler(
  target: DetectTarget,
  label: string,
  onInterfacesLoaded: (interfaces: NetworkInterface[]) => void,
  onChange: (value: string) => void,
  setToast: (toast: ToastState) => void,
): () => Promise<void> {
  return async () => {
    const interfaces = await fetchJson<NetworkInterface[]>("/api/interfaces");
    onInterfacesLoaded(interfaces);
    const response = await fetchJson<DetectResponse>(
      `/api/autodetect-${target}`,
      { method: "POST" },
    );
    const detectedIface = response.iface ?? "";
    onChange(detectedIface);
    setToast({
      kind: "success",
      message: detectedIface
        ? `Auto-detected ${label}: ${detectedIface}`
        : `No working ${label} found.`,
    });
  };
}

export function GeneralSettingsPanel({
  form,
  gfwProxyActive,
  interfaceOptions,
  isLoading,
  onInterfacesLoaded,
  setField,
  setToast,
}: GeneralSettingsPanelProps) {
  return (
    <section className="panel">
      <header className="panel__header">General Settings</header>
      <div className="panel__body">
        <div className="field">
          <label htmlFor="socks5-port">SOCKS5 Port</label>
          <input
            disabled={isLoading}
            id="socks5-port"
            inputMode="numeric"
            name="port"
            onChange={(event) => {
              setField("port", event.target.value);
            }}
            type="number"
            value={form.port}
          />
        </div>

        <InterfaceSelector
          disabled={isLoading}
          label="Default Interface"
          onChange={(val) => setField("defaultIface", val)}
          options={interfaceOptions}
          value={form.defaultIface}
        />

        <InterfaceSelector
          disabled={isLoading || gfwProxyActive}
          label="GFW Interface (Personal VPN)"
          onChange={(val) => setField("gfwIface", val)}
          onDetect={makeDetectHandler(
            "gfw",
            "GFW Interface (Personal VPN)",
            onInterfacesLoaded,
            (val) => setField("gfwIface", val),
            setToast,
          )}
          onDetectError={(error) =>
            setToast({
              kind: "error",
              message:
                error instanceof Error ? error.message : "Auto-detect failed.",
            })
          }
          options={interfaceOptions}
          value={form.gfwIface}
          warningMessage={
            gfwProxyActive ? "Overridden by upstream proxy." : undefined
          }
        />

        <InterfaceSelector
          disabled={isLoading}
          label="Company Interface (Company VPN)"
          onChange={(val) => setField("companyIface", val)}
          onDetect={makeDetectHandler(
            "company",
            "Company Interface (Company VPN)",
            onInterfacesLoaded,
            (val) => setField("companyIface", val),
            setToast,
          )}
          onDetectError={(error) =>
            setToast({
              kind: "error",
              message:
                error instanceof Error ? error.message : "Auto-detect failed.",
            })
          }
          options={interfaceOptions}
          value={form.companyIface}
        />
      </div>
    </section>
  );
}

type HttpProxyPanelProps = {
  form: FormState;
  httpProxyText: string;
  interfaceOptions: NetworkInterface[];
  isLoading: boolean;
  setField: FieldSetter;
};

export function HttpProxyPanel({
  form,
  httpProxyText,
  interfaceOptions,
  isLoading,
  setField,
}: HttpProxyPanelProps) {
  return (
    <section className="panel">
      <header className="panel__header">HTTP Proxy</header>
      <div className="panel__body">
        <InterfaceSelector
          disabled={isLoading}
          label="Outgoing Interface"
          onChange={(val) => setField("httpProxyIface", val)}
          options={interfaceOptions}
          value={form.httpProxyIface}
        />

        <div className="status-card">
          HTTP proxy port: <strong>{httpProxyText}</strong>
        </div>
      </div>
    </section>
  );
}

type RulesSettingsPanelProps = {
  form: FormState;
  isLoading: boolean;
  isUpdatingGFWList: boolean;
  onUpdateGFWList: () => void;
  setField: FieldSetter;
};

export function RulesSettingsPanel({
  form,
  isLoading,
  isUpdatingGFWList,
  onUpdateGFWList,
  setField,
}: RulesSettingsPanelProps) {
  return (
    <section className="panel">
      <header className="panel__header">Rules &amp; Settings</header>
      <div className="panel__body">
        <div className="field">
          <label htmlFor="company-domains">Company Domains</label>
          <TokenInput
            describedBy="company-domains-description"
            disabled={isLoading}
            id="company-domains"
            onChange={(nextDomains) => {
              setField("companyDomains", nextDomains);
            }}
            placeholder="e.g. company.com"
            tokens={form.companyDomains}
          />
          <p className="field-description" id="company-domains-description">
            Domains routed through Company VPN. Use comma, Enter, or paste to
            add domains.
          </p>
        </div>

        <div className="field">
          <label htmlFor="bypass-domains">Bypass Domains (Direct)</label>
          <TokenInput
            describedBy="bypass-domains-description"
            disabled={isLoading}
            id="bypass-domains"
            onChange={(nextDomains) => {
              setField("bypassDomains", nextDomains);
            }}
            placeholder="e.g. example.com"
            tokens={form.bypassDomains}
          />
          <p className="field-description" id="bypass-domains-description">
            Domains that always bypass the proxy. Use comma, Enter, or paste to
            add domains.
          </p>
        </div>

        <div className="field">
          <label htmlFor="extra-gfw-domains">Extra GFW Domains</label>
          <TokenInput
            describedBy="extra-gfw-domains-description"
            disabled={isLoading}
            id="extra-gfw-domains"
            onChange={(nextDomains) => {
              setField("extraGfwDomains", nextDomains);
            }}
            placeholder="e.g. google.com"
            tokens={form.extraGfwDomains}
          />
          <p className="field-description" id="extra-gfw-domains-description">
            Domains forced through the GFW route. Use comma, Enter, or paste to
            add domains.
          </p>
        </div>

        <div className="field">
          <label htmlFor="gfwlist-url">GFWList URL/Path</label>
          <input
            disabled={isLoading}
            id="gfwlist-url"
            name="gfwlistUrl"
            onChange={(event) => {
              setField("gfwlistUrl", event.target.value);
            }}
            type="text"
            value={form.gfwlistUrl}
          />
        </div>

        <div className="field">
          <button
            disabled={isLoading || isUpdatingGFWList}
            onClick={onUpdateGFWList}
            type="button"
          >
            {isUpdatingGFWList
              ? "Checking GFWList…"
              : "Check for GFWList Updates"}
          </button>
          <p className="field-description">
            Checks the official GFWList and downloads it when a newer version is
            available.
          </p>
        </div>
      </div>
    </section>
  );
}

import TokenInput from "./TokenInput";

import type { DetectTarget, FormState, NetworkInterface } from "../types";

type FieldSetter = <K extends keyof FormState>(
  key: K,
  value: FormState[K],
) => void;

type QuickSettingsPanelProps = {
  detectedSystemProxy: string;
  form: FormState;
  isRefreshingSystemProxy: boolean;
  onRefreshSystemProxy: () => void;
  setField: FieldSetter;
  setUseDetectedSystemProxy: (value: boolean) => void;
  useDetectedSystemProxy: boolean;
};

export function QuickSettingsPanel({
  detectedSystemProxy,
  form,
  isRefreshingSystemProxy,
  onRefreshSystemProxy,
  setField,
  setUseDetectedSystemProxy,
  useDetectedSystemProxy,
}: QuickSettingsPanelProps) {
  return (
    <section className="panel">
      <header className="panel__header">Quick Settings</header>
      <div className="panel__body">
        <label className="switch" htmlFor="auto-start-toggle">
          <input
            checked={form.autoStart}
            id="auto-start-toggle"
            onChange={(event) => {
              setField("autoStart", event.target.checked);
            }}
            type="checkbox"
          />
          <span>Auto-start proxy on program launch</span>
        </label>
        <label className="switch" htmlFor="verbose-log-toggle">
          <input
            checked={form.verboseLog}
            id="verbose-log-toggle"
            onChange={(event) => {
              setField("verboseLog", event.target.checked);
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
  detectingTargets: Set<DetectTarget>;
  form: FormState;
  gfwProxyActive: boolean;
  interfaceOptions: NetworkInterface[];
  isLoading: boolean;
  onAutoDetect: (target: DetectTarget) => void;
  setField: FieldSetter;
};

export function GeneralSettingsPanel({
  detectingTargets,
  form,
  gfwProxyActive,
  interfaceOptions,
  isLoading,
  onAutoDetect,
  setField,
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

        <div className="field">
          <label htmlFor="default-interface">Default Interface</label>
          <select
            disabled={isLoading}
            id="default-interface"
            name="defaultIface"
            onChange={(event) => {
              setField("defaultIface", event.target.value);
            }}
            value={form.defaultIface}
          >
            {interfaceOptions.map((option) => (
              <option key={option.name || "none"} value={option.name}>
                {option.name || "None"}
              </option>
            ))}
          </select>
        </div>

        <div className="field">
          <label htmlFor="gfw-interface">GFW Interface (Personal VPN)</label>
          <div className="inline-control">
            <select
              aria-describedby={
                gfwProxyActive ? "gfw-interface-hint" : undefined
              }
              disabled={isLoading || gfwProxyActive}
              id="gfw-interface"
              name="gfwIface"
              onChange={(event) => {
                setField("gfwIface", event.target.value);
              }}
              value={form.gfwIface}
            >
              {interfaceOptions.map((option) => (
                <option
                  key={`gfw-${option.name || "none"}`}
                  value={option.name}
                >
                  {option.name || "None"}
                </option>
              ))}
            </select>
            <button
              className="button button--secondary button--outline"
              disabled={detectingTargets.has("gfw") || gfwProxyActive}
              onClick={() => {
                onAutoDetect("gfw");
              }}
              type="button"
            >
              {detectingTargets.has("gfw") ? "Testing..." : "🔍 Detect"}
            </button>
          </div>
          {gfwProxyActive ? (
            <small
              className="field-hint field-hint--warning"
              id="gfw-interface-hint"
            >
              Overridden by upstream proxy.
            </small>
          ) : null}
        </div>

        <div className="field">
          <label htmlFor="company-interface">
            Company Interface (Company VPN)
          </label>
          <div className="inline-control">
            <select
              disabled={isLoading}
              id="company-interface"
              name="companyIface"
              onChange={(event) => {
                setField("companyIface", event.target.value);
              }}
              value={form.companyIface}
            >
              {interfaceOptions.map((option) => (
                <option
                  key={`company-${option.name || "none"}`}
                  value={option.name}
                >
                  {option.name || "None"}
                </option>
              ))}
            </select>
            <button
              className="button button--secondary button--outline"
              disabled={detectingTargets.has("company")}
              onClick={() => {
                onAutoDetect("company");
              }}
              type="button"
            >
              {detectingTargets.has("company") ? "Testing..." : "🔍 Detect"}
            </button>
          </div>
        </div>
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
        <div className="field">
          <label htmlFor="http-proxy-interface">Outgoing Interface</label>
          <select
            disabled={isLoading}
            id="http-proxy-interface"
            name="httpProxyIface"
            onChange={(event) => {
              setField("httpProxyIface", event.target.value);
            }}
            value={form.httpProxyIface}
          >
            {interfaceOptions.map((option) => (
              <option key={`http-${option.name || "none"}`} value={option.name}>
                {option.name || "None"}
              </option>
            ))}
          </select>
        </div>

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
  setField: FieldSetter;
};

export function RulesSettingsPanel({
  form,
  isLoading,
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
      </div>
    </section>
  );
}

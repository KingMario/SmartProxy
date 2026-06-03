import { useEffect, useMemo, useRef, useState } from "react";

import "./App.scss";

import AppHeader from "./components/AppHeader";
import LogsPanel from "./components/LogsPanel";
import {
  GeneralSettingsPanel,
  HttpProxyPanel,
  QuickSettingsPanel,
  RulesSettingsPanel,
} from "./components/SettingsPanels";
import type {
  ConfigResponse,
  ControlAction,
  DetectResponse,
  DetectTarget,
  FormState,
  NetworkInterface,
  StatusResponse,
  SystemProxyResponse,
  ToastState,
} from "./types";
import {
  buildConfigPayload,
  createFormState,
  defaultFormState,
  defaultStatus,
  fetchJson,
  getConfigSignature,
  normalizeDomains,
  normalizeStatus,
  postWithoutResponse,
} from "./utils";

function App() {
  const [form, setForm] = useState<FormState>(defaultFormState);
  const [interfaces, setInterfaces] = useState<NetworkInterface[]>([]);
  const [status, setStatus] = useState<StatusResponse>(defaultStatus);
  const [savedConfigSignature, setSavedConfigSignature] = useState<
    string | null
  >(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isClearingLogs, setIsClearingLogs] = useState(false);
  const [isRefreshingSystemProxy, setIsRefreshingSystemProxy] = useState(false);
  const [controlAction, setControlAction] = useState<ControlAction | null>(
    null,
  );
  const [detectingTargets, setDetectingTargets] = useState<Set<DetectTarget>>(
    new Set(),
  );
  const [useDetectedSystemProxy, setUseDetectedSystemProxy] = useState(false);
  const [systemProxyHint, setSystemProxyHint] = useState("");
  const [toast, setToast] = useState<ToastState>(null);
  const logRef = useRef<HTMLDivElement | null>(null);

  const detectedSystemProxy = systemProxyHint.trim();
  const effectiveGfwProxy =
    useDetectedSystemProxy && detectedSystemProxy ? detectedSystemProxy : "";
  const gfwProxyActive = effectiveGfwProxy.trim().length > 0;
  const visibleLogs = useMemo(() => status.logs ?? [], [status.logs]);
  const currentConfigSignature = useMemo(
    () => getConfigSignature(buildConfigPayload(form, effectiveGfwProxy)),
    [effectiveGfwProxy, form],
  );
  const hasUnsavedChanges =
    !isLoading &&
    savedConfigSignature !== null &&
    currentConfigSignature !== savedConfigSignature;
  const interfaceOptions = useMemo(
    () => [{ index: 0, name: "" }, ...interfaces],
    [interfaces],
  );
  const httpProxyText = status.httpProxyPort
    ? `${status.httpProxyIface || "HTTP"} -> 0.0.0.0:${status.httpProxyPort}`
    : "Starting...";
  const statusText = status.running
    ? `● Running (0.0.0.0:${status.port || 1080})`
    : "○ Stopped";

  useEffect(() => {
    const timeoutId = window.setTimeout(() => {
      setToast(null);
    }, 3000);

    if (!toast) {
      window.clearTimeout(timeoutId);
    }

    return () => {
      window.clearTimeout(timeoutId);
    };
  }, [toast]);

  useEffect(() => {
    if (!visibleLogs.length || !logRef.current) {
      return;
    }

    logRef.current.scrollTop = logRef.current.scrollHeight;
  }, [visibleLogs]);

  useEffect(() => {
    const loadData = async () => {
      try {
        const [loadedInterfaces, config, systemProxy] = await Promise.all([
          fetchJson<NetworkInterface[]>("/api/interfaces"),
          fetchJson<ConfigResponse>("/api/config"),
          fetchJson<SystemProxyResponse>("/api/detect-system-proxy").catch(
            () => ({
              proxy: "",
            }),
          ),
        ]);

        setInterfaces(loadedInterfaces);
        setForm(createFormState(config));
        setSystemProxyHint(systemProxy.proxy ?? "");
        setUseDetectedSystemProxy(
          Boolean(
            systemProxy.proxy &&
            config.gfwProxy &&
            config.gfwProxy === systemProxy.proxy,
          ),
        );
        setSavedConfigSignature(
          getConfigSignature({
            autoStart: config.autoStart ?? false,
            bypassDomains: normalizeDomains(config.bypassDomains ?? []),
            companyDomains: normalizeDomains(config.companyDomains ?? []),
            companyIface: config.companyIface ?? "",
            defaultIface: config.defaultIface ?? "",
            extraGfwDomains: normalizeDomains(config.extraGfwDomains ?? []),
            gfwIface: config.gfwIface ?? "",
            gfwProxy: config.gfwProxy ?? "",
            gfwlistUrl: config.gfwlistUrl ?? "",
            httpProxyIface: config.httpProxyIface ?? "",
            port: config.port || 1080,
            verboseLog: config.verboseLog ?? false,
          }),
        );
      } catch (error) {
        setToast({
          kind: "error",
          message:
            error instanceof Error ? error.message : "Failed to load data.",
        });
      } finally {
        setIsLoading(false);
      }
    };

    void loadData();
  }, []);

  useEffect(() => {
    const updateStatus = async () => {
      try {
        const nextStatus = await fetchJson<StatusResponse>("/api/status");
        setStatus(normalizeStatus(nextStatus));
      } catch (error) {
        setToast(
          (currentToast) =>
            currentToast ?? {
              kind: "error",
              message:
                error instanceof Error
                  ? error.message
                  : "Failed to refresh status.",
            },
        );
      }
    };

    void updateStatus();
    const intervalId = window.setInterval(() => {
      void updateStatus();
    }, 1000);

    return () => {
      window.clearInterval(intervalId);
    };
  }, []);

  const setField = <K extends keyof FormState>(key: K, value: FormState[K]) => {
    setForm((currentForm) => ({
      ...currentForm,
      [key]: value,
    }));
  };

  const refreshInterfaces = async () => {
    const loadedInterfaces =
      await fetchJson<NetworkInterface[]>("/api/interfaces");
    setInterfaces(loadedInterfaces);
  };

  const refreshSystemProxyState = async () => {
    setIsRefreshingSystemProxy(true);

    try {
      const systemProxy = await fetchJson<SystemProxyResponse>(
        "/api/detect-system-proxy",
      ).catch(() => ({
        proxy: "",
      }));

      const nextProxy = (systemProxy.proxy ?? "").trim();
      setSystemProxyHint(nextProxy);
      setUseDetectedSystemProxy((currentValue) =>
        nextProxy ? currentValue : false,
      );
    } finally {
      setIsRefreshingSystemProxy(false);
    }
  };

  const handleSave = async () => {
    setIsSaving(true);

    try {
      const configPayload = buildConfigPayload(form, effectiveGfwProxy);
      await postWithoutResponse("/api/config", configPayload);
      setSavedConfigSignature(getConfigSignature(configPayload));
      setToast({
        kind: "success",
        message: "Configuration saved successfully.",
      });
    } catch (error) {
      setToast({
        kind: "error",
        message:
          error instanceof Error
            ? error.message
            : "Failed to save configuration.",
      });
    } finally {
      setIsSaving(false);
    }
  };

  const handleControl = async (action: ControlAction) => {
    setControlAction(action);

    try {
      await postWithoutResponse(`/api/${action}`);
      const nextStatus = await fetchJson<StatusResponse>("/api/status");
      setStatus(normalizeStatus(nextStatus));
    } catch (error) {
      setToast({
        kind: "error",
        message:
          error instanceof Error ? error.message : `Failed to ${action} proxy.`,
      });
    } finally {
      setControlAction(null);
    }
  };

  const handleAutoDetect = async (target: DetectTarget) => {
    setDetectingTargets((prev) => new Set(prev).add(target));

    try {
      await refreshInterfaces();
      const response = await fetchJson<DetectResponse>(
        `/api/autodetect-${target}`,
        {
          method: "POST",
        },
      );
      const detectedIface = response.iface ?? "";

      if (target === "gfw") {
        setField("gfwIface", detectedIface);
      } else {
        setField("companyIface", detectedIface);
      }

      setToast({
        kind: "success",
        message: detectedIface
          ? `Auto-detected ${target === "gfw" ? "GFW" : "Company"} Interface: ${detectedIface}`
          : `No working ${target === "gfw" ? "GFW" : "Company"} interface found.`,
      });
    } catch (error) {
      setToast({
        kind: "error",
        message: error instanceof Error ? error.message : "Auto-detect failed.",
      });
    } finally {
      setDetectingTargets((prev) => {
        const next = new Set(prev);
        next.delete(target);
        return next;
      });
    }
  };

  const handleClearLogs = async () => {
    setIsClearingLogs(true);

    try {
      await postWithoutResponse("/api/logs/clear");
      setStatus((currentStatus) => ({
        ...currentStatus,
        logs: [],
      }));
    } catch (error) {
      setToast({
        kind: "error",
        message:
          error instanceof Error ? error.message : "Failed to clear logs.",
      });
    } finally {
      setIsClearingLogs(false);
    }
  };

  useEffect(() => {
    const handleKeydown = (event: KeyboardEvent) => {
      const key = event.key.toLowerCase();

      if ((event.metaKey || event.ctrlKey) && key === "s") {
        event.preventDefault();
        void handleSave();
      }

      if ((event.metaKey || event.ctrlKey) && !event.shiftKey && key === "r") {
        event.preventDefault();
        void handleControl("restart");
      }
    };

    document.addEventListener("keydown", handleKeydown);

    return () => {
      document.removeEventListener("keydown", handleKeydown);
    };
  });

  useEffect(() => {
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      if (!hasUnsavedChanges) {
        return;
      }

      event.preventDefault();
      event.returnValue = "";
    };

    window.addEventListener("beforeunload", handleBeforeUnload);

    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [hasUnsavedChanges]);

  return (
    <div className="app-shell">
      <a className="skip-link" href="#main-content">
        Skip to main content
      </a>
      <main className="app-inner" id="main-content" tabIndex={-1}>
        <AppHeader
          controlAction={controlAction}
          hasUnsavedChanges={hasUnsavedChanges}
          isSaving={isSaving}
          onControl={(action) => {
            void handleControl(action);
          }}
          onSave={() => {
            void handleSave();
          }}
          running={status.running}
          statusText={statusText}
        />

        <div className="app-grid">
          <div className="settings-column">
            <QuickSettingsPanel
              detectedSystemProxy={detectedSystemProxy}
              form={form}
              isRefreshingSystemProxy={isRefreshingSystemProxy}
              onRefreshSystemProxy={() => {
                void refreshSystemProxyState();
              }}
              setField={setField}
              setUseDetectedSystemProxy={setUseDetectedSystemProxy}
              useDetectedSystemProxy={useDetectedSystemProxy}
            />
            <GeneralSettingsPanel
              detectingTargets={detectingTargets}
              form={form}
              gfwProxyActive={gfwProxyActive}
              interfaceOptions={interfaceOptions}
              isLoading={isLoading}
              onAutoDetect={(target) => {
                void handleAutoDetect(target);
              }}
              setField={setField}
            />
            <HttpProxyPanel
              form={form}
              httpProxyText={httpProxyText}
              interfaceOptions={interfaceOptions}
              isLoading={isLoading}
              setField={setField}
            />
            <RulesSettingsPanel
              form={form}
              isLoading={isLoading}
              setField={setField}
            />
          </div>

          <LogsPanel
            isClearingLogs={isClearingLogs}
            logRef={logRef}
            onClearLogs={() => {
              void handleClearLogs();
            }}
            visibleLogs={visibleLogs}
          />
        </div>
      </main>

      {toast ? (
        <div
          aria-atomic="true"
          aria-live="polite"
          className={`toast toast--${toast.kind}`}
          role="status"
        >
          {toast.message}
        </div>
      ) : null}
    </div>
  );
}

export default App;

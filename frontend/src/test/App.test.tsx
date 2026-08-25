import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  render,
  screen,
  act,
  fireEvent,
  waitFor,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import App from "../App";
import type { ConfigResponse } from "../types";

vi.mock("../utils", async (importOriginal) => {
  const original = await importOriginal<typeof import("../utils")>();
  return {
    ...original,
    fetchJson: vi.fn(),
    postWithoutResponse: vi.fn(),
  };
});

import { fetchJson, postWithoutResponse } from "../utils";
const mockFetchJson = fetchJson as ReturnType<typeof vi.fn>;
const mockPost = postWithoutResponse as ReturnType<typeof vi.fn>;

const baseConfig: ConfigResponse = {
  autoStart: false,
  autoUpdateGfwList: false,
  bypassDomains: [],
  companyDomains: [],
  companyIface: "",
  defaultIface: "",
  extraGfwDomains: [],
  gfwIface: "",
  gfwProxy: "",
  gfwlistUrl: "",
  httpProxyIface: "",
  port: 1080,
  verboseLog: false,
};

function setupMocks(configOverride?: Partial<ConfigResponse>) {
  const config = { ...baseConfig, ...configOverride };
  mockFetchJson.mockImplementation((url: string) => {
    if (url === "/api/interfaces") return Promise.resolve([]);
    if (url === "/api/config") return Promise.resolve(config);
    if (url.includes("detect-system-proxy"))
      return Promise.resolve({ proxy: "" });
    if (url === "/api/status")
      return Promise.resolve({ running: false, logs: [], port: 1080 });
    return Promise.reject(new Error(`Unexpected URL: ${url}`));
  });
  mockPost.mockResolvedValue(undefined);
}

// delay: null disables userEvent's internal setTimeout waits
const user = userEvent.setup({ delay: null });

async function renderApp() {
  render(<App />);
  // Wait for initial data load to settle
  await screen.findByRole("spinbutton", { name: /socks5 port/i });
}

describe("App", () => {
  beforeEach(() => {
    setupMocks();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  // ── Initial render ────────────────────────────────────────────────────────

  it("shows stopped status on initial render", async () => {
    await renderApp();
    expect(screen.getByText(/○ Stopped/)).toBeInTheDocument();
  });

  it("loads and displays port from config", async () => {
    await renderApp();
    expect(
      screen.getByRole("spinbutton", { name: /socks5 port/i }),
    ).toHaveValue(1080);
  });

  it("shows 'Starting...' for HTTP proxy when no port is reported", async () => {
    await renderApp();
    expect(screen.getByText("Starting...")).toBeInTheDocument();
  });

  it("shows error toast when initial data load fails", async () => {
    mockFetchJson.mockRejectedValue(new Error("Network error"));
    render(<App />);
    expect(await screen.findByRole("status")).toHaveTextContent(
      "Network error",
    );
  });

  // ── Unsaved changes ───────────────────────────────────────────────────────

  it("does not show 'Unsaved changes' right after load", async () => {
    await renderApp();
    expect(screen.queryByText("Unsaved changes")).not.toBeInTheDocument();
  });

  it("shows 'Unsaved changes' after modifying a form field", async () => {
    await renderApp();
    const portInput = screen.getByRole("spinbutton", { name: /socks5 port/i });
    await user.clear(portInput);
    await user.type(portInput, "9090");
    expect(screen.getByText("Unsaved changes")).toBeInTheDocument();
  });

  it("clears 'Unsaved changes' after saving", async () => {
    await renderApp();
    const portInput = screen.getByRole("spinbutton", { name: /socks5 port/i });
    await user.clear(portInput);
    await user.type(portInput, "9090");
    await user.click(screen.getByLabelText("Save"));
    await waitFor(() =>
      expect(screen.queryByText("Unsaved changes")).not.toBeInTheDocument(),
    );
  });

  // ── Save ──────────────────────────────────────────────────────────────────

  it("calls postWithoutResponse with config when Save is clicked", async () => {
    await renderApp();
    await user.click(screen.getByLabelText("Save"));
    await waitFor(() =>
      expect(mockPost).toHaveBeenCalledWith(
        "/api/config",
        expect.objectContaining({ port: 1080 }),
      ),
    );
  });

  it("shows success toast after successful save", async () => {
    await renderApp();
    await user.click(screen.getByLabelText("Save"));
    expect(await screen.findByRole("status")).toHaveTextContent(
      "Configuration saved successfully.",
    );
  });

  it("shows error toast when save fails", async () => {
    mockPost.mockRejectedValue(new Error("Save failed"));
    await renderApp();
    await user.click(screen.getByLabelText("Save"));
    expect(await screen.findByRole("status")).toHaveTextContent("Save failed");
  });

  it("disables Save button while saving", async () => {
    let resolveSave!: () => void;
    mockPost.mockReturnValue(
      new Promise<void>((r) => {
        resolveSave = r;
      }),
    );
    await renderApp();
    await user.click(screen.getByLabelText("Save"));
    expect(screen.getByLabelText("Saving")).toBeDisabled();
    resolveSave();
    await waitFor(() =>
      expect(screen.getByLabelText("Save")).not.toBeDisabled(),
    );
  });

  // ── Control ───────────────────────────────────────────────────────────────

  it("calls postWithoutResponse('/api/start') when Start is clicked", async () => {
    await renderApp();
    await user.click(screen.getByLabelText("Start"));
    await waitFor(() => expect(mockPost).toHaveBeenCalledWith("/api/start"));
  });

  it("shows error toast when control action fails", async () => {
    mockPost.mockRejectedValue(new Error("Start failed"));
    await renderApp();
    await user.click(screen.getByLabelText("Start"));
    expect(await screen.findByRole("status")).toHaveTextContent("Start failed");
  });

  // ── Clear logs ────────────────────────────────────────────────────────────

  it("calls postWithoutResponse('/api/logs/clear') when Clear is clicked", async () => {
    await renderApp();
    await user.click(screen.getByRole("button", { name: "Clear" }));
    await waitFor(() =>
      expect(mockPost).toHaveBeenCalledWith("/api/logs/clear"),
    );
  });

  it("shows error toast when clear logs fails", async () => {
    mockPost.mockRejectedValue(new Error("Clear failed"));
    await renderApp();
    await user.click(screen.getByRole("button", { name: "Clear" }));
    expect(await screen.findByRole("status")).toHaveTextContent("Clear failed");
  });

  // ── Keyboard shortcuts ────────────────────────────────────────────────────

  it("triggers save on Cmd+S", async () => {
    await renderApp();
    fireEvent.keyDown(document, { key: "s", metaKey: true });
    await waitFor(() =>
      expect(mockPost).toHaveBeenCalledWith("/api/config", expect.anything()),
    );
  });

  it("triggers restart on Cmd+R", async () => {
    await renderApp();
    fireEvent.keyDown(document, { key: "r", metaKey: true });
    await waitFor(() => expect(mockPost).toHaveBeenCalledWith("/api/restart"));
  });

  it("does not trigger restart on Cmd+Shift+R", async () => {
    await renderApp();
    fireEvent.keyDown(document, { key: "r", metaKey: true, shiftKey: true });
    // Give it a moment to not fire
    await act(async () => {});
    expect(mockPost).not.toHaveBeenCalledWith("/api/restart");
  });

  // ── Toast auto-dismiss ────────────────────────────────────────────────────

  it("auto-dismisses toast after 3 seconds", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: false });
    try {
      setupMocks();
      mockPost.mockRejectedValue(new Error("err"));

      render(<App />);
      // Flush initial load (Promises resolve as microtasks, no real timers needed)
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });

      // Trigger save via fireEvent (no internal setTimeout unlike userEvent)
      fireEvent.keyDown(document, { key: "s", metaKey: true });
      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });

      expect(screen.getByRole("status")).toBeInTheDocument();

      await act(async () => {
        await vi.advanceTimersByTimeAsync(3000);
      });
      expect(screen.queryByRole("status")).not.toBeInTheDocument();
    } finally {
      vi.clearAllTimers();
      vi.useRealTimers();
    }
  });

  // ── System proxy ──────────────────────────────────────────────────────────

  it("shows detected system proxy address", async () => {
    mockFetchJson.mockImplementation((url: string) => {
      if (url === "/api/interfaces") return Promise.resolve([]);
      if (url === "/api/config") return Promise.resolve(baseConfig);
      if (url.includes("detect-system-proxy"))
        return Promise.resolve({ proxy: "http://127.0.0.1:7890" });
      if (url === "/api/status")
        return Promise.resolve({ running: false, logs: [], port: 1080 });
      return Promise.reject(new Error(`Unexpected: ${url}`));
    });
    await renderApp();
    expect(screen.getByText(/http:\/\/127\.0\.0\.1:7890/)).toBeInTheDocument();
  });

  it("refreshes system proxy when Refresh is clicked", async () => {
    await renderApp();
    await user.click(screen.getByRole("button", { name: "Refresh" }));
    await waitFor(() =>
      expect(mockFetchJson).toHaveBeenCalledWith("/api/detect-system-proxy"),
    );
  });

  // ── Status display ────────────────────────────────────────────────────────

  it("shows running status when proxy is running", async () => {
    mockFetchJson.mockImplementation((url: string) => {
      if (url === "/api/interfaces") return Promise.resolve([]);
      if (url === "/api/config") return Promise.resolve(baseConfig);
      if (url.includes("detect-system-proxy"))
        return Promise.resolve({ proxy: "" });
      if (url === "/api/status")
        return Promise.resolve({ running: true, logs: [], port: 1080 });
      return Promise.reject(new Error(`Unexpected: ${url}`));
    });
    await renderApp();
    expect(screen.getByText(/● Running/)).toBeInTheDocument();
  });

  it("shows HTTP proxy text when port is available", async () => {
    mockFetchJson.mockImplementation((url: string) => {
      if (url === "/api/interfaces") return Promise.resolve([]);
      if (url === "/api/config") return Promise.resolve(baseConfig);
      if (url.includes("detect-system-proxy"))
        return Promise.resolve({ proxy: "" });
      if (url === "/api/status")
        return Promise.resolve({
          running: true,
          logs: [],
          port: 1080,
          httpProxyPort: 8080,
          httpProxyIface: "eth0",
        });
      return Promise.reject(new Error(`Unexpected: ${url}`));
    });
    await renderApp();
    expect(screen.getByText(/eth0 -> 0\.0\.0\.0:8080/)).toBeInTheDocument();
  });
});

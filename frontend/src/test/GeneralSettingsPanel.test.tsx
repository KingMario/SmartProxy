import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GeneralSettingsPanel } from "../components/SettingsPanels";
import { defaultFormState } from "../utils";
import type { FormState, NetworkInterface } from "../types";

vi.mock("../utils", async (importOriginal) => {
  const original = await importOriginal<typeof import("../utils")>();
  return {
    ...original,
    fetchJson: vi.fn(),
  };
});

import { fetchJson } from "../utils";
const mockFetchJson = fetchJson as ReturnType<typeof vi.fn>;

const interfaces: NetworkInterface[] = [
  { index: 0, name: "" },
  { index: 1, name: "eth0" },
  { index: 2, name: "tun0" },
];

const baseForm: FormState = {
  ...defaultFormState,
  gfwIface: "",
  companyIface: "",
};

const baseProps = {
  form: baseForm,
  gfwProxyActive: false,
  interfaceOptions: interfaces,
  isLoading: false,
  onInterfacesLoaded: vi.fn(),
  setField: vi.fn(),
  setToast: vi.fn(),
};

describe("GeneralSettingsPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders SOCKS5 port input with current value", () => {
    render(<GeneralSettingsPanel {...baseProps} form={{ ...baseForm, port: "9090" }} />);
    expect(screen.getByRole("spinbutton", { name: /socks5 port/i })).toHaveValue(9090);
  });

  it("calls setField('port', ...) when port input changes", async () => {
    const setField = vi.fn();
    render(<GeneralSettingsPanel {...baseProps} setField={setField} />);
    const input = screen.getByRole("spinbutton", { name: /socks5 port/i });
    await userEvent.clear(input);
    await userEvent.type(input, "9");
    expect(setField).toHaveBeenCalledWith("port", expect.stringContaining("9"));
  });

  it("renders default interface select with all options", () => {
    render(<GeneralSettingsPanel {...baseProps} />);
    const noneOptions = screen.getAllByRole("option", { name: "None" });
    expect(noneOptions.length).toBeGreaterThanOrEqual(1);
    const eth0Options = screen.getAllByRole("option", { name: "eth0" });
    expect(eth0Options.length).toBeGreaterThanOrEqual(1);
  });

  it("calls setField('defaultIface', ...) when default interface changes", async () => {
    const setField = vi.fn();
    render(<GeneralSettingsPanel {...baseProps} setField={setField} />);
    const selects = screen.getAllByRole("combobox");
    await userEvent.selectOptions(selects[0], "eth0");
    expect(setField).toHaveBeenCalledWith("defaultIface", "eth0");
  });

  it("disables all inputs while isLoading", () => {
    render(<GeneralSettingsPanel {...baseProps} isLoading />);
    for (const el of screen.getAllByRole("combobox")) {
      expect(el).toBeDisabled();
    }
    expect(screen.getByRole("spinbutton")).toBeDisabled();
  });

  it("shows GFW override warning when gfwProxyActive", () => {
    render(<GeneralSettingsPanel {...baseProps} gfwProxyActive />);
    expect(screen.getByText("Overridden by upstream proxy.")).toBeInTheDocument();
  });

  it("does not show GFW override warning when gfwProxyActive is false", () => {
    render(<GeneralSettingsPanel {...baseProps} gfwProxyActive={false} />);
    expect(
      screen.queryByText("Overridden by upstream proxy."),
    ).not.toBeInTheDocument();
  });

  it("calls onInterfacesLoaded and setField('gfwIface') on successful GFW detect", async () => {
    const onInterfacesLoaded = vi.fn();
    const setField = vi.fn();
    const setToast = vi.fn();

    mockFetchJson
      .mockResolvedValueOnce(interfaces)
      .mockResolvedValueOnce({ iface: "tun0" });

    render(
      <GeneralSettingsPanel
        {...baseProps}
        onInterfacesLoaded={onInterfacesLoaded}
        setField={setField}
        setToast={setToast}
      />,
    );

    const detectButtons = screen.getAllByRole("button", { name: /detect/i });
    await userEvent.click(detectButtons[0]);

    expect(onInterfacesLoaded).toHaveBeenCalledWith(interfaces);
    expect(setField).toHaveBeenCalledWith("gfwIface", "tun0");
    expect(setToast).toHaveBeenCalledWith(
      expect.objectContaining({ kind: "success" }),
    );
  });

  it("calls setToast with error when GFW detect fails", async () => {
    const setToast = vi.fn();
    mockFetchJson.mockRejectedValue(new Error("Network timeout"));

    render(<GeneralSettingsPanel {...baseProps} setToast={setToast} />);

    const detectButtons = screen.getAllByRole("button", { name: /detect/i });
    await userEvent.click(detectButtons[0]);

    expect(
      await screen.findAllByRole("button", { name: /detect/i }),
    ).not.toHaveLength(0);
    expect(setToast).toHaveBeenCalledWith(
      expect.objectContaining({ kind: "error", message: "Network timeout" }),
    );
  });

  it("calls setField('companyIface') on successful Company detect", async () => {
    const setField = vi.fn();
    mockFetchJson
      .mockResolvedValueOnce(interfaces)
      .mockResolvedValueOnce({ iface: "eth0" });

    render(<GeneralSettingsPanel {...baseProps} setField={setField} />);

    const detectButtons = screen.getAllByRole("button", { name: /detect/i });
    await userEvent.click(detectButtons[1]);

    expect(setField).toHaveBeenCalledWith("companyIface", "eth0");
  });

  it("calls setField('gfwIface', ...) when GFW interface is manually selected", async () => {
    const setField = vi.fn();
    render(<GeneralSettingsPanel {...baseProps} setField={setField} />);
    const selects = screen.getAllByRole("combobox");
    await userEvent.selectOptions(selects[1], "tun0");
    expect(setField).toHaveBeenCalledWith("gfwIface", "tun0");
  });

  it("calls setField('companyIface', ...) when Company interface is manually selected", async () => {
    const setField = vi.fn();
    render(<GeneralSettingsPanel {...baseProps} setField={setField} />);
    const selects = screen.getAllByRole("combobox");
    await userEvent.selectOptions(selects[2], "eth0");
    expect(setField).toHaveBeenCalledWith("companyIface", "eth0");
  });

  it("calls setToast with error when Company detect fails", async () => {
    const setToast = vi.fn();
    mockFetchJson.mockRejectedValue(new Error("Company timeout"));

    render(<GeneralSettingsPanel {...baseProps} setToast={setToast} />);

    const detectButtons = screen.getAllByRole("button", { name: /detect/i });
    await userEvent.click(detectButtons[1]);

    expect(
      await screen.findAllByRole("button", { name: /detect/i }),
    ).not.toHaveLength(0);
    expect(setToast).toHaveBeenCalledWith(
      expect.objectContaining({ kind: "error", message: "Company timeout" }),
    );
  });

  it("shows 'No working interface found' toast when detect returns empty iface", async () => {
    const setToast = vi.fn();
    mockFetchJson
      .mockResolvedValueOnce(interfaces)
      .mockResolvedValueOnce({ iface: "" });

    render(<GeneralSettingsPanel {...baseProps} setToast={setToast} />);

    const detectButtons = screen.getAllByRole("button", { name: /detect/i });
    await userEvent.click(detectButtons[0]);

    expect(setToast).toHaveBeenCalledWith(
      expect.objectContaining({
        kind: "success",
        message: expect.stringContaining("No working"),
      }),
    );
  });
});

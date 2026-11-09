import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { HttpProxyPanel } from "../components/SettingsPanels";
import { defaultFormState } from "../utils";

const interfaces = [
  { index: 0, name: "" },
  { index: 1, name: "eth0" },
];

const baseProps = {
  form: defaultFormState,
  httpProxyText: "0.0.0.0:17891",
  interfaceOptions: interfaces,
  isLoading: false,
  setField: vi.fn(),
};

describe("HttpProxyPanel", () => {
  it("edits the dedicated interface DNS servers", async () => {
    const setField = vi.fn();
    render(<HttpProxyPanel {...baseProps} setField={setField} />);
    await userEvent.type(screen.getByLabelText("Interface DNS servers (optional)"), "1");
    expect(setField).toHaveBeenCalledWith("httpProxyDNS", "1");
  });
  it("renders the HTTP proxy status text", () => {
    render(<HttpProxyPanel {...baseProps} />);
    expect(screen.getByText("0.0.0.0:17891")).toBeInTheDocument();
    expect(screen.queryByText("0.0.0.0:17890")).not.toBeInTheDocument();
  });

  it("renders interface options", () => {
    render(<HttpProxyPanel {...baseProps} />);
    expect(screen.getByRole("option", { name: "None" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "eth0" })).toBeInTheDocument();
  });

  it("reflects current httpProxyIface value", () => {
    render(
      <HttpProxyPanel
        {...baseProps}
        form={{ ...defaultFormState, httpProxyIface: "eth0" }}
      />,
    );
    expect(screen.getByRole("combobox")).toHaveValue("eth0");
  });

  it("calls setField('httpProxyIface', ...) when interface changes", async () => {
    const setField = vi.fn();
    render(<HttpProxyPanel {...baseProps} setField={setField} />);
    await userEvent.selectOptions(screen.getByRole("combobox"), "eth0");
    expect(setField).toHaveBeenCalledWith("httpProxyIface", "eth0");
  });

  it("disables interface select while isLoading", () => {
    render(<HttpProxyPanel {...baseProps} isLoading />);
    expect(screen.getByRole("combobox")).toBeDisabled();
  });
});

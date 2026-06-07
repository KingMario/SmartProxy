import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RulesSettingsPanel } from "../components/SettingsPanels";
import { defaultFormState } from "../utils";

const baseProps = {
  form: defaultFormState,
  isLoading: false,
  setField: vi.fn(),
};

describe("RulesSettingsPanel", () => {
  it("renders company domains tokens", () => {
    render(
      <RulesSettingsPanel
        {...baseProps}
        form={{ ...defaultFormState, companyDomains: ["corp.com"] }}
      />,
    );
    expect(screen.getByText("corp.com")).toBeInTheDocument();
  });

  it("calls setField('companyDomains', ...) when company domain is added", async () => {
    const setField = vi.fn();
    render(<RulesSettingsPanel {...baseProps} setField={setField} />);
    const inputs = screen.getAllByRole("textbox");
    await userEvent.type(inputs[0], "corp.com{Enter}");
    expect(setField).toHaveBeenCalledWith("companyDomains", ["corp.com"]);
  });

  it("calls setField('bypassDomains', ...) when bypass domain is added", async () => {
    const setField = vi.fn();
    render(<RulesSettingsPanel {...baseProps} setField={setField} />);
    const inputs = screen.getAllByRole("textbox");
    await userEvent.type(inputs[1], "direct.com{Enter}");
    expect(setField).toHaveBeenCalledWith("bypassDomains", ["direct.com"]);
  });

  it("calls setField('extraGfwDomains', ...) when extra GFW domain is added", async () => {
    const setField = vi.fn();
    render(<RulesSettingsPanel {...baseProps} setField={setField} />);
    const inputs = screen.getAllByRole("textbox");
    await userEvent.type(inputs[2], "google.com{Enter}");
    expect(setField).toHaveBeenCalledWith("extraGfwDomains", ["google.com"]);
  });

  it("reflects gfwlistUrl value", () => {
    render(
      <RulesSettingsPanel
        {...baseProps}
        form={{ ...defaultFormState, gfwlistUrl: "https://example.com/gfw.txt" }}
      />,
    );
    expect(screen.getByRole("textbox", { name: /gfwlist url/i })).toHaveValue(
      "https://example.com/gfw.txt",
    );
  });

  it("calls setField('gfwlistUrl', ...) when GFWList URL changes", async () => {
    const setField = vi.fn();
    render(<RulesSettingsPanel {...baseProps} setField={setField} />);
    const urlInput = screen.getByRole("textbox", { name: /gfwlist url/i });
    await userEvent.click(urlInput);
    await userEvent.keyboard("x");
    expect(setField).toHaveBeenCalledWith("gfwlistUrl", "x");
  });

  it("disables all inputs while isLoading", () => {
    render(<RulesSettingsPanel {...baseProps} isLoading />);
    for (const el of screen.getAllByRole("textbox")) {
      expect(el).toBeDisabled();
    }
  });
});

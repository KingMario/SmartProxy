import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QuickSettingsPanel } from "../components/SettingsPanels";

const baseProps = {
  autoStart: false,
  verboseLog: false,
  detectedSystemProxy: "",
  isRefreshingSystemProxy: false,
  onAutoStartChange: vi.fn(),
  onVerboseLogChange: vi.fn(),
  onRefreshSystemProxy: vi.fn(),
  setUseDetectedSystemProxy: vi.fn(),
  useDetectedSystemProxy: false,
};

describe("QuickSettingsPanel", () => {
  it("reflects autoStart checked state", () => {
    render(<QuickSettingsPanel {...baseProps} autoStart />);
    expect(screen.getByLabelText(/auto-start proxy/i)).toBeChecked();
  });

  it("calls onAutoStartChange when auto-start toggle is changed", async () => {
    const onAutoStartChange = vi.fn();
    render(
      <QuickSettingsPanel {...baseProps} onAutoStartChange={onAutoStartChange} />,
    );
    await userEvent.click(screen.getByLabelText(/auto-start proxy/i));
    expect(onAutoStartChange).toHaveBeenCalledWith(true);
  });

  it("reflects verboseLog checked state", () => {
    render(<QuickSettingsPanel {...baseProps} verboseLog />);
    expect(screen.getByLabelText(/verbose logging/i)).toBeChecked();
  });

  it("calls onVerboseLogChange when verbose-log toggle is changed", async () => {
    const onVerboseLogChange = vi.fn();
    render(
      <QuickSettingsPanel {...baseProps} onVerboseLogChange={onVerboseLogChange} />,
    );
    await userEvent.click(screen.getByLabelText(/verbose logging/i));
    expect(onVerboseLogChange).toHaveBeenCalledWith(true);
  });

  it("shows detected proxy address when proxy is detected", () => {
    render(
      <QuickSettingsPanel
        {...baseProps}
        detectedSystemProxy="http://127.0.0.1:7890"
      />,
    );
    expect(screen.getByText(/http:\/\/127\.0\.0\.1:7890/)).toBeInTheDocument();
  });

  it("shows 'not detected' message when no proxy is detected", () => {
    render(<QuickSettingsPanel {...baseProps} detectedSystemProxy="" />);
    expect(screen.getByText(/not detected/i)).toBeInTheDocument();
  });

  it("disables proxy toggle when no system proxy is detected", () => {
    render(<QuickSettingsPanel {...baseProps} detectedSystemProxy="" />);
    expect(screen.getByRole("checkbox", { name: /global proxy/i })).toBeDisabled();
  });

  it("enables proxy toggle when system proxy is detected", () => {
    render(
      <QuickSettingsPanel
        {...baseProps}
        detectedSystemProxy="http://127.0.0.1:7890"
      />,
    );
    expect(
      screen.getByRole("checkbox", { name: /global proxy/i }),
    ).toBeEnabled();
  });

  it("calls onRefreshSystemProxy when Refresh is clicked", async () => {
    const onRefreshSystemProxy = vi.fn();
    render(
      <QuickSettingsPanel
        {...baseProps}
        onRefreshSystemProxy={onRefreshSystemProxy}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: "Refresh" }));
    expect(onRefreshSystemProxy).toHaveBeenCalled();
  });

  it("calls setUseDetectedSystemProxy when proxy toggle is changed", async () => {
    const setUseDetectedSystemProxy = vi.fn();
    render(
      <QuickSettingsPanel
        {...baseProps}
        detectedSystemProxy="http://127.0.0.1:7890"
        setUseDetectedSystemProxy={setUseDetectedSystemProxy}
      />,
    );
    await userEvent.click(screen.getByRole("checkbox", { name: /global proxy/i }));
    expect(setUseDetectedSystemProxy).toHaveBeenCalledWith(true);
  });
});

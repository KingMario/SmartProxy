import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import AppHeader from "../components/AppHeader";

const baseProps = {
  controlAction: null,
  hasUnsavedChanges: false,
  isSaving: false,
  onControl: vi.fn(),
  onSave: vi.fn(),
  running: false,
  statusText: "○ Stopped",
};

describe("AppHeader", () => {
  it("renders status text", () => {
    render(<AppHeader {...baseProps} statusText="● Running (0.0.0.0:1080)" />);
    expect(screen.getByText("● Running (0.0.0.0:1080)")).toBeInTheDocument();
  });

  it("shows 'Unsaved changes' badge when hasUnsavedChanges is true", () => {
    render(<AppHeader {...baseProps} hasUnsavedChanges />);
    expect(screen.getByText("Unsaved changes")).toBeInTheDocument();
  });

  it("hides 'Unsaved changes' badge when hasUnsavedChanges is false", () => {
    render(<AppHeader {...baseProps} hasUnsavedChanges={false} />);
    expect(screen.queryByText("Unsaved changes")).not.toBeInTheDocument();
  });

  it("disables Start button when proxy is running", () => {
    render(<AppHeader {...baseProps} running />);
    expect(screen.getByLabelText("Start")).toBeDisabled();
  });

  it("enables Start button when proxy is stopped", () => {
    render(<AppHeader {...baseProps} running={false} />);
    expect(screen.getByLabelText("Start")).toBeEnabled();
  });

  it("disables Stop and Restart when proxy is stopped", () => {
    render(<AppHeader {...baseProps} running={false} />);
    expect(screen.getByLabelText("Stop")).toBeDisabled();
    expect(screen.getByLabelText("Restart")).toBeDisabled();
  });

  it("enables Stop and Restart when proxy is running", () => {
    render(<AppHeader {...baseProps} running />);
    expect(screen.getByLabelText("Stop")).toBeEnabled();
    expect(screen.getByLabelText("Restart")).toBeEnabled();
  });

  it("disables all control buttons while a control action is in progress", () => {
    render(<AppHeader {...baseProps} running controlAction="restart" />);
    expect(screen.getByLabelText("Start")).toBeDisabled();
    expect(screen.getByLabelText("Stop")).toBeDisabled();
    expect(screen.getByLabelText("Restart")).toBeDisabled();
  });

  it("calls onControl('start') when Start is clicked", async () => {
    const onControl = vi.fn();
    render(<AppHeader {...baseProps} onControl={onControl} />);
    await userEvent.click(screen.getByLabelText("Start"));
    expect(onControl).toHaveBeenCalledWith("start");
  });

  it("calls onControl('stop') when Stop is clicked", async () => {
    const onControl = vi.fn();
    render(<AppHeader {...baseProps} running onControl={onControl} />);
    await userEvent.click(screen.getByLabelText("Stop"));
    expect(onControl).toHaveBeenCalledWith("stop");
  });

  it("calls onControl('restart') when Restart is clicked", async () => {
    const onControl = vi.fn();
    render(<AppHeader {...baseProps} running onControl={onControl} />);
    await userEvent.click(screen.getByLabelText("Restart"));
    expect(onControl).toHaveBeenCalledWith("restart");
  });

  it("calls onSave when Save is clicked", async () => {
    const onSave = vi.fn();
    render(<AppHeader {...baseProps} onSave={onSave} />);
    await userEvent.click(screen.getByLabelText("Save"));
    expect(onSave).toHaveBeenCalled();
  });

  it("disables Save and shows saving label while isSaving", () => {
    render(<AppHeader {...baseProps} isSaving />);
    expect(screen.getByLabelText("Saving")).toBeDisabled();
  });
});

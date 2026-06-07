import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import LogsPanel from "../components/LogsPanel";
import { createRef } from "react";

const baseProps = {
  isClearingLogs: false,
  logRef: createRef<HTMLDivElement>(),
  onClearLogs: vi.fn(),
  visibleLogs: [],
};

describe("LogsPanel", () => {
  it("renders each log line", () => {
    render(<LogsPanel {...baseProps} visibleLogs={["line 1", "line 2"]} />);
    expect(screen.getByText("line 1")).toBeInTheDocument();
    expect(screen.getByText("line 2")).toBeInTheDocument();
  });

  it("renders nothing in the log console when logs are empty", () => {
    const { container } = render(<LogsPanel {...baseProps} visibleLogs={[]} />);
    expect(container.querySelector(".log-console")?.children).toHaveLength(0);
  });

  it("calls onClearLogs when Clear is clicked", async () => {
    const onClearLogs = vi.fn();
    render(<LogsPanel {...baseProps} onClearLogs={onClearLogs} />);
    await userEvent.click(screen.getByRole("button", { name: "Clear" }));
    expect(onClearLogs).toHaveBeenCalled();
  });

  it("disables Clear button while isClearingLogs", () => {
    render(<LogsPanel {...baseProps} isClearingLogs />);
    expect(screen.getByRole("button", { name: "Clearing..." })).toBeDisabled();
  });
});

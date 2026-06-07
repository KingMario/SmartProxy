import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { InterfaceSelector } from "../components/InterfaceSelector";

const options = [
  { index: 0, name: "" },
  { index: 1, name: "eth0" },
  { index: 2, name: "tun0" },
];

const baseProps = {
  label: "GFW Interface",
  value: "",
  options,
  onChange: vi.fn(),
  onDetect: vi.fn().mockResolvedValue(undefined),
  disabled: false,
};

describe("InterfaceSelector", () => {
  it("renders the label", () => {
    render(<InterfaceSelector {...baseProps} />);
    expect(screen.getByText("GFW Interface")).toBeInTheDocument();
  });

  it("renders all options", () => {
    render(<InterfaceSelector {...baseProps} />);
    expect(screen.getByRole("option", { name: "None" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "eth0" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "tun0" })).toBeInTheDocument();
  });

  it("reflects selected value", () => {
    render(<InterfaceSelector {...baseProps} value="eth0" />);
    expect(screen.getByRole("combobox")).toHaveValue("eth0");
  });

  it("calls onChange when select value changes", async () => {
    const onChange = vi.fn();
    render(<InterfaceSelector {...baseProps} onChange={onChange} />);
    await userEvent.selectOptions(screen.getByRole("combobox"), "eth0");
    expect(onChange).toHaveBeenCalledWith("eth0");
  });

  it("disables select and Detect button when disabled", () => {
    render(<InterfaceSelector {...baseProps} disabled />);
    expect(screen.getByRole("combobox")).toBeDisabled();
    expect(screen.getByRole("button", { name: /detect/i })).toBeDisabled();
  });

  it("shows warningMessage when provided", () => {
    render(
      <InterfaceSelector
        {...baseProps}
        warningMessage="Overridden by upstream proxy."
      />,
    );
    expect(
      screen.getByText("Overridden by upstream proxy."),
    ).toBeInTheDocument();
  });

  it("does not render warning element when warningMessage is absent", () => {
    render(<InterfaceSelector {...baseProps} />);
    expect(screen.queryByRole("note")).not.toBeInTheDocument();
  });

  it("links select to warning via aria-describedby when warningMessage is present", () => {
    render(
      <InterfaceSelector
        {...baseProps}
        warningMessage="Overridden by upstream proxy."
      />,
    );
    const select = screen.getByRole("combobox");
    const hintId = select.getAttribute("aria-describedby");
    expect(hintId).toBeTruthy();
    expect(document.getElementById(hintId!)).toHaveTextContent(
      "Overridden by upstream proxy.",
    );
  });

  it("calls onDetect when Detect button is clicked", async () => {
    const onDetect = vi.fn().mockResolvedValue(undefined);
    render(<InterfaceSelector {...baseProps} onDetect={onDetect} />);
    await userEvent.click(screen.getByRole("button", { name: /detect/i }));
    expect(onDetect).toHaveBeenCalled();
  });

  it("shows 'Testing...' and disables Detect button while onDetect is pending", async () => {
    let resolve!: () => void;
    const onDetect = vi.fn(
      () => new Promise<void>((r) => { resolve = r; }),
    );
    render(<InterfaceSelector {...baseProps} onDetect={onDetect} />);
    await userEvent.click(screen.getByRole("button", { name: /detect/i }));
    expect(
      await screen.findByRole("button", { name: "Testing..." }),
    ).toBeDisabled();
    resolve();
  });

  it("re-enables Detect button after onDetect resolves", async () => {
    const onDetect = vi.fn().mockResolvedValue(undefined);
    render(<InterfaceSelector {...baseProps} onDetect={onDetect} />);
    await userEvent.click(screen.getByRole("button", { name: /detect/i }));
    expect(
      await screen.findByRole("button", { name: /detect/i }),
    ).toBeEnabled();
  });

  it("re-enables Detect button after onDetect rejects", async () => {
    const onDetect = vi.fn().mockRejectedValue(new Error("Network error"));
    const onDetectError = vi.fn();
    render(
      <InterfaceSelector
        {...baseProps}
        onDetect={onDetect}
        onDetectError={onDetectError}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: /detect/i }));
    expect(
      await screen.findByRole("button", { name: /detect/i }),
    ).toBeEnabled();
    expect(onDetectError).toHaveBeenCalledWith(expect.any(Error));
  });

  it("does not render Detect button when onDetect is not provided", () => {
    const { onDetect: _omit, ...propsWithoutDetect } = baseProps;
    render(<InterfaceSelector {...propsWithoutDetect} />);
    expect(
      screen.queryByRole("button", { name: /detect/i }),
    ).not.toBeInTheDocument();
  });

  it("renders select without inline-control button when onDetect is absent", () => {
    const { onDetect: _omit, ...propsWithoutDetect } = baseProps;
    render(<InterfaceSelector {...propsWithoutDetect} />);
    expect(screen.getByRole("combobox")).toBeInTheDocument();
  });
});

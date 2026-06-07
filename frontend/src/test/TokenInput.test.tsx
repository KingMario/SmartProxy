import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import TokenInput from "../components/TokenInput";

const baseProps = {
  id: "test-input",
  tokens: [],
  onChange: vi.fn(),
};

describe("TokenInput", () => {
  it("renders existing tokens", () => {
    render(<TokenInput {...baseProps} tokens={["example.com", "foo.com"]} />);
    expect(screen.getByText("example.com")).toBeInTheDocument();
    expect(screen.getByText("foo.com")).toBeInTheDocument();
  });

  it("shows placeholder when tokens list is empty", () => {
    render(
      <TokenInput {...baseProps} tokens={[]} placeholder="e.g. domain.com" />,
    );
    expect(screen.getByPlaceholderText("e.g. domain.com")).toBeInTheDocument();
  });

  it("hides placeholder when tokens exist", () => {
    render(
      <TokenInput
        {...baseProps}
        tokens={["a.com"]}
        placeholder="e.g. domain.com"
      />,
    );
    expect(
      screen.queryByPlaceholderText("e.g. domain.com"),
    ).not.toBeInTheDocument();
  });

  it("calls onChange with new token when Enter is pressed", async () => {
    const onChange = vi.fn();
    render(<TokenInput {...baseProps} onChange={onChange} />);
    const input = screen.getByRole("textbox");
    await userEvent.type(input, "new.com{Enter}");
    expect(onChange).toHaveBeenCalledWith(["new.com"]);
  });

  it("calls onChange with new token when comma is pressed", async () => {
    const onChange = vi.fn();
    render(<TokenInput {...baseProps} onChange={onChange} />);
    await userEvent.type(screen.getByRole("textbox"), "new.com,");
    expect(onChange).toHaveBeenCalledWith(["new.com"]);
  });

  it("calls onChange with new token on blur", async () => {
    const onChange = vi.fn();
    render(<TokenInput {...baseProps} onChange={onChange} />);
    const input = screen.getByRole("textbox");
    await userEvent.type(input, "new.com");
    await userEvent.tab();
    expect(onChange).toHaveBeenCalledWith(["new.com"]);
  });

  it("does not call onChange when committing blank input", async () => {
    const onChange = vi.fn();
    render(<TokenInput {...baseProps} onChange={onChange} />);
    await userEvent.type(screen.getByRole("textbox"), "   {Enter}");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("appends new token to existing tokens", async () => {
    const onChange = vi.fn();
    render(
      <TokenInput
        {...baseProps}
        tokens={["existing.com"]}
        onChange={onChange}
      />,
    );
    await userEvent.type(screen.getByRole("textbox"), "new.com{Enter}");
    expect(onChange).toHaveBeenCalledWith(["existing.com", "new.com"]);
  });

  it("removes token when its Remove button is clicked", async () => {
    const onChange = vi.fn();
    render(
      <TokenInput
        {...baseProps}
        tokens={["a.com", "b.com"]}
        onChange={onChange}
      />,
    );
    await userEvent.click(screen.getByLabelText("Remove a.com"));
    expect(onChange).toHaveBeenCalledWith(["b.com"]);
  });

  it("removes last token on Backspace when input is empty", async () => {
    const onChange = vi.fn();
    render(
      <TokenInput
        {...baseProps}
        tokens={["a.com", "b.com"]}
        onChange={onChange}
      />,
    );
    await userEvent.type(screen.getByRole("textbox"), "{Backspace}");
    expect(onChange).toHaveBeenCalledWith(["a.com"]);
  });

  it("does not remove token on Backspace when input has text", async () => {
    const onChange = vi.fn();
    render(
      <TokenInput {...baseProps} tokens={["a.com"]} onChange={onChange} />,
    );
    await userEvent.type(screen.getByRole("textbox"), "t{Backspace}");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("splits pasted text containing commas into multiple tokens", async () => {
    const onChange = vi.fn();
    render(<TokenInput {...baseProps} onChange={onChange} />);
    await userEvent.click(screen.getByRole("textbox"));
    await userEvent.paste("a.com,b.com,c.com");
    expect(onChange).toHaveBeenCalledWith(["a.com", "b.com", "c.com"]);
  });

  it("splits pasted text containing newlines into multiple tokens", async () => {
    const onChange = vi.fn();
    render(<TokenInput {...baseProps} onChange={onChange} />);
    await userEvent.click(screen.getByRole("textbox"));
    await userEvent.paste("a.com\nb.com");
    expect(onChange).toHaveBeenCalledWith(["a.com", "b.com"]);
  });

  it("deduplicates tokens on paste", async () => {
    const onChange = vi.fn();
    render(
      <TokenInput
        {...baseProps}
        tokens={["a.com"]}
        onChange={onChange}
      />,
    );
    await userEvent.click(screen.getByRole("textbox"));
    await userEvent.paste("a.com,b.com");
    expect(onChange).toHaveBeenCalledWith(["a.com", "b.com"]);
  });

  it("disables input and remove buttons when disabled", () => {
    render(
      <TokenInput
        {...baseProps}
        tokens={["a.com"]}
        disabled
      />,
    );
    expect(screen.getByRole("textbox")).toBeDisabled();
    expect(screen.getByLabelText("Remove a.com")).toBeDisabled();
  });
});

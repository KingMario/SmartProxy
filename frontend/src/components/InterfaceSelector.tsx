import { useState } from "react";
import type { NetworkInterface } from "../types";

type InterfaceSelectorProps = {
  label: string;
  value: string;
  options: NetworkInterface[];
  onChange: (value: string) => void;
  onDetect?: () => Promise<void>;
  onDetectError?: (error: unknown) => void;
  disabled: boolean;
  warningMessage?: string;
};

export function InterfaceSelector({
  label,
  value,
  options,
  onChange,
  onDetect,
  onDetectError,
  disabled,
  warningMessage,
}: InterfaceSelectorProps) {
  const [isDetecting, setIsDetecting] = useState(false);
  const id = `${label.toLowerCase().replace(/\s+/g, "-")}-interface`;
  const hintId = warningMessage ? `${id}-hint` : undefined;

  const handleDetect = async () => {
    if (!onDetect) return;
    setIsDetecting(true);
    try {
      await onDetect();
    } catch (error) {
      onDetectError?.(error);
    } finally {
      setIsDetecting(false);
    }
  };

  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <div className="inline-control">
        <select
          aria-describedby={hintId}
          disabled={disabled}
          id={id}
          onChange={(e) => onChange(e.target.value)}
          value={value}
        >
          {options.map((option) => (
            <option key={`${id}-${option.name || "none"}`} value={option.name}>
              {option.name || "None"}
            </option>
          ))}
        </select>
        {onDetect ? (
          <button
            className="button button--secondary button--outline"
            disabled={disabled || isDetecting}
            onClick={() => {
              void handleDetect();
            }}
            type="button"
          >
            {isDetecting ? "Testing..." : "🔍 Detect"}
          </button>
        ) : null}
      </div>
      {warningMessage ? (
        <small className="field-hint field-hint--warning" id={hintId}>
          {warningMessage}
        </small>
      ) : null}
    </div>
  );
}

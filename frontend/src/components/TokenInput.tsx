import { useRef, useState } from "react";

import { normalizeDomains, parseDomainTokens } from "../utils";

type TokenInputProps = {
  describedBy?: string;
  disabled?: boolean;
  id: string;
  onChange: (tokens: string[]) => void;
  placeholder?: string;
  tokens: string[];
};

function TokenInput({
  describedBy,
  disabled = false,
  id,
  onChange,
  placeholder,
  tokens,
}: TokenInputProps) {
  const [draft, setDraft] = useState("");
  const inputRef = useRef<HTMLInputElement | null>(null);

  const commitDraft = (value: string) => {
    const nextTokens = parseDomainTokens(value);

    if (!nextTokens.length) {
      setDraft("");
      return;
    }

    onChange(normalizeDomains([...tokens, ...nextTokens]));
    setDraft("");
  };

  const removeToken = (tokenToRemove: string) => {
    onChange(tokens.filter((token) => token !== tokenToRemove));
  };

  return (
    <div
      className={`token-input ${disabled ? "token-input--disabled" : ""}`}
      onClick={() => {
        inputRef.current?.focus();
      }}
    >
      {tokens.map((token) => (
        <span className="token-input__token" key={token}>
          <span>{token}</span>
          <button
            aria-label={`Remove ${token}`}
            className="token-input__remove"
            disabled={disabled}
            onClick={(event) => {
              event.stopPropagation();
              removeToken(token);
            }}
            type="button"
          >
            ×
          </button>
        </span>
      ))}
      <input
        aria-describedby={describedBy}
        className="token-input__field"
        disabled={disabled}
        id={id}
        onBlur={() => {
          commitDraft(draft);
        }}
        onChange={(event) => {
          setDraft(event.target.value);
        }}
        onKeyDown={(event) => {
          if (
            (event.key === "Enter" ||
              event.key === "," ||
              event.key === "Tab") &&
            draft.trim()
          ) {
            event.preventDefault();
            commitDraft(draft);
            return;
          }

          if (event.key === "Backspace" && !draft && tokens.length) {
            event.preventDefault();
            removeToken(tokens[tokens.length - 1]);
          }
        }}
        onPaste={(event) => {
          const pastedText = event.clipboardData.getData("text");

          if (/[,\n]/.test(pastedText)) {
            event.preventDefault();
            commitDraft(pastedText);
          }
        }}
        placeholder={tokens.length ? "" : placeholder}
        ref={inputRef}
        type="text"
        value={draft}
      />
    </div>
  );
}

export default TokenInput;

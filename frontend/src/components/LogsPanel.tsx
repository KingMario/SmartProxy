import type { RefObject } from "react";

type LogsPanelProps = {
  isClearingLogs: boolean;
  logRef: RefObject<HTMLDivElement | null>;
  onClearLogs: () => void;
  visibleLogs: string[];
};

function LogsPanel({
  isClearingLogs,
  logRef,
  onClearLogs,
  visibleLogs,
}: LogsPanelProps) {
  return (
    <section className="panel logs-panel">
      <header className="panel__header panel__header--split">
        <span>Real-time Logs</span>
        <button
          className="button button--danger button--small button--outline"
          disabled={isClearingLogs}
          onClick={onClearLogs}
          type="button"
        >
          {isClearingLogs ? "Clearing..." : "Clear"}
        </button>
      </header>
      <div className="panel__body panel__body--flush">
        <div aria-live="polite" className="log-console" ref={logRef} role="log">
          {visibleLogs.map((line, index) => (
            <div key={`${line}-${index}`}>{line}</div>
          ))}
        </div>
      </div>
    </section>
  );
}

export default LogsPanel;

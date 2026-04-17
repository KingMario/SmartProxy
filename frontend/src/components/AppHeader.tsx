import type { ControlAction } from "../types";

type AppHeaderProps = {
  controlAction: ControlAction | null;
  hasUnsavedChanges: boolean;
  isSaving: boolean;
  onControl: (action: ControlAction) => void;
  onSave: () => void;
  running: boolean;
  statusText: string;
};

function AppHeader({
  controlAction,
  hasUnsavedChanges,
  isSaving,
  onControl,
  onSave,
  running,
  statusText,
}: AppHeaderProps) {
  return (
    <header className="app-header">
      <div className="app-title-group">
        <h1>🚀 Smart Proxy</h1>
        <div className="toolbar" role="group" aria-label="Proxy controls">
          <button
            aria-label="Start"
            className="button button--ghost"
            disabled={running || controlAction !== null}
            onClick={() => {
              onControl("start");
            }}
            title="Start"
            type="button"
          >
            ▶️
          </button>
          <button
            aria-label="Restart"
            className="button button--warning"
            disabled={!running || controlAction !== null}
            onClick={() => {
              onControl("restart");
            }}
            title="Restart"
            type="button"
          >
            🔄
          </button>
          <button
            aria-label="Stop"
            className="button button--danger"
            disabled={!running || controlAction !== null}
            onClick={() => {
              onControl("stop");
            }}
            title="Stop"
            type="button"
          >
            ⏹️
          </button>
          <button
            aria-label={isSaving ? "Saving" : "Save"}
            className="button button--primary"
            disabled={isSaving}
            onClick={onSave}
            title={isSaving ? "Saving" : "Save"}
            type="button"
          >
            {isSaving ? "⏳" : "💾"}
          </button>
        </div>
      </div>
      <div className="header-actions">
        {hasUnsavedChanges ? (
          <div
            aria-live="polite"
            className="status-badge status-badge--pending"
          >
            Unsaved changes
          </div>
        ) : null}
        <div
          aria-live="polite"
          className={`status-badge ${running ? "status-badge--on" : "status-badge--off"}`}
        >
          {statusText}
        </div>
      </div>
    </header>
  );
}

export default AppHeader;

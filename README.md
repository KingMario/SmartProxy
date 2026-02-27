# 🚀 SmartProxy

**SmartProxy** is a cross-platform network utility (macOS + Windows) designed to solve the complexity of managing multiple network environments simultaneously. It acts as an intelligent router that sits between your applications and your various network interfaces (Company VPN, Personal VPN/GFW, and Direct/Local), allowing seamless access to all resources without manual switching.

## 🌟 Key Features

*   **Intelligent Routing**: Automatically routes traffic based on domain rules.
    *   **Company Domains**: Routes specified corporate domains through your Company VPN interface.
    *   **GFW List**: Automatically routes blocked domains (via `gfwlist.txt` + custom rules) through your Personal VPN interface.
    *   **Direct/Bypass**: Keeps local and regular traffic on your default interface for maximum speed.
*   **Modern Web GUI**: A clean, responsive Bootstrap-based control panel to manage settings and view real-time logs.
*   **System Tray Integration**:
    *   Quick "Start/Stop" controls from the system tray.
    *   One-click access to the configuration page.
    *   Visual status indicator (🚀).
*   **Zero-Conflict Architecture**:
    *   Uses a **random available port** for the GUI to prevent "Address already in use" errors.
    *   **Single Instance Lock** ensures you don't accidentally run multiple copies.
*   **Developer Friendly**:
    *   Real-time connection logging for debugging network paths.
    *   JSON-based configuration for easy backup/restore.
    *   One-click build script (`build.sh`) included.

## 🛠 Installation & Build

### Prerequisites
*   Go 1.21+
*   macOS (for `.app` bundle) or Windows (for `.exe`)

### Building from Source

1.  Clone the repository:
    ```bash
    git clone https://github.com/MarioStudio/SmartProxy.git
    cd SmartProxy
    ```

2.  Build the macOS application bundle:
    ```bash
    ./build.sh
    ```
    This will create/update `SmartProxy.app` in the current directory.

### Build for Windows

Use one of the following methods:

1.  Via script (recommended):
    ```bash
    ./build-windows.sh
    ```

    This script automatically installs `github.com/akavel/rsrc`, embeds `assets/tray.ico` into a temporary `SmartProxy.syso`, and then builds the Windows GUI binary with that icon.

2.  Or direct Go command (manual icon embedding):
    ```bash
    go install github.com/akavel/rsrc@v0.10.2
    rsrc -ico assets/tray.ico -o SmartProxy.syso
    GOOS=windows GOARCH=amd64 go build -ldflags="-H=windowsgui" -o SmartProxy.exe .
    rm -f SmartProxy.syso
    ```

This will generate `SmartProxy.exe` (no console window) with the same core proxy + tray + web control panel workflow, and includes the tray icon from `assets/tray.ico`.

### Build All Targets

Use one command to build all available targets:

```bash
./build-all.sh
```

Output behavior:
*   Always builds:
    *   `SmartProxy.exe` (Windows GUI, no console)
*   Builds `SmartProxy.app` only when running on a macOS host; on Linux/Windows hosts it is skipped with a message.

### Build Windows Installer

To generate an installer (`SmartProxy-Installer.exe`), make sure the prerequisites below are satisfied, then use Inno Setup on a Windows machine.

#### Prerequisites
*   `SmartProxy.exe` already exists (run `./build-windows.sh` first so the icon-embedded GUI binary is ready).
*   **Inno Setup 6** installed and `ISCC` available in PATH (the script auto-detects `%ProgramFiles(x86)%\Inno Setup 6\ISCC.exe`).

1. Install **Inno Setup 6**.
2. In the project root, run:
   ```bat
   build-installer.bat
   ```

Generated file:
*   `dist\windows\SmartProxy-Installer.exe`

Installer script location:
*   `installer/windows/SmartProxy.iss`

## 📖 Usage

1.  **Launch the App**: On macOS, open `SmartProxy.app`; on Windows, run `SmartProxy.exe`.
2.  **Open Configuration**:
    *   Click the 🚀 icon in the system tray and select **Open Configuration**.
    *   Or, check the logs for the GUI URL (e.g., `http://127.0.0.1:54321`).
3.  **Setup Interfaces**:
    *   **Default Interface**: Your main internet connection (e.g., `en0`).
    *   **GFW Interface**: Your personal VPN's virtual interface (e.g., `utun6`).
    *   **Company Interface**: Your corporate VPN's interface (e.g., `utun7`).
4.  **Configure Rules**:
    *   Add company domains to the **Company Domains** list.
    *   Add custom blocked sites to **Extra GFW Domains**.
    *   Hit **Save** (or `Cmd+S`) to apply changes immediately.

## 📝 License

Copyright © 2026-2027 MarioStudio.
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

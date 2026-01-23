# 🚀 SmartProxy

**SmartProxy** is a macOS-exclusive network utility designed to solve the complexity of managing multiple network environments simultaneously. It acts as an intelligent router that sits between your applications and your various network interfaces (Company VPN, Personal VPN/GFW, and Direct/Local), allowing seamless access to all resources without manual switching.

## 🌟 Key Features

*   **Intelligent Routing**: Automatically routes traffic based on domain rules.
    *   **Company Domains**: Routes specified corporate domains through your Company VPN interface.
    *   **GFW List**: Automatically routes blocked domains (via `gfwlist.txt` + custom rules) through your Personal VPN interface.
    *   **Direct/Bypass**: Keeps local and regular traffic on your default interface for maximum speed.
*   **Modern Web GUI**: A clean, responsive Bootstrap-based control panel to manage settings and view real-time logs.
*   **System Tray Integration**:
    *   Quick "Start/Stop" controls from the macOS menu bar.
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
*   macOS (Darwin)
*   Go 1.21+

### Building from Source

1.  Clone the repository:
    ```bash
    git clone https://github.com/MarioStudio/SmartProxy.git
    cd SmartProxy
    ```

2.  Build the application bundle:
    ```bash
    ./build.sh
    ```
    This will create/update `SmartProxy.app` in the current directory.

## 📖 Usage

1.  **Launch the App**: Double-click `SmartProxy.app` or run it from the terminal.
2.  **Open Configuration**:
    *   Click the 🚀 icon in the menu bar and select **Open Configuration**.
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

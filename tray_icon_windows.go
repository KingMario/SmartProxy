//go:build windows

package main

import _ "embed"

import "github.com/getlantern/systray"

//go:embed assets/tray.ico
var windowsTrayIcon []byte

func setPlatformTrayIcon() {
	if len(windowsTrayIcon) > 0 {
		systray.SetIcon(windowsTrayIcon)
	}
}

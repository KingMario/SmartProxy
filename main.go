package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/getlantern/systray"
)

func main() {
	guiPort := flag.Int("gui-port", 0, "Port for GUI console (0 for random)")
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".smart-proxy")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		os.MkdirAll(configDir, 0755)
	}

	// Single instance check
	lockFile := filepath.Join(configDir, "smart-proxy.lock")
	releaseLock, err := acquireInstanceLock(lockFile)
	if err != nil {
		if err == ErrAlreadyRunning {
			fmt.Println("Another instance of Smart Proxy is already running.")
			os.Exit(0)
		}
		fmt.Printf("Error acquiring lock file: %v\n", err)
		os.Exit(1)
	}

	defaultConfigPath := filepath.Join(configDir, "config.json")
	configPath := flag.String("config", defaultConfigPath, "Path to config file")
	flag.Parse()

	logFile, err := os.OpenFile(filepath.Join(configDir, "output.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		mw := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(mw)
	}

	p := &ProxyServer{
		configPath: *configPath,
		Config: Config{
			Port:           1080,
			DefaultIface:   "en0",
			GFWListURL:     filepath.Join(configDir, "gfwlist.txt"),
			AutoStart:      true,
			HTTPProxyIface: "en0",
			GFWProxy:       "",
		},
	}

	// Ensure gfwlist.txt exists in configDir
	gfwDest := filepath.Join(configDir, "gfwlist.txt")
	if _, err := os.Stat(gfwDest); os.IsNotExist(err) {
		execPath, _ := os.Executable()
		searchPaths := []string{
			filepath.Join(filepath.Dir(execPath), "gfwlist.txt"),
			filepath.Join(filepath.Dir(execPath), "..", "Resources", "gfwlist.txt"),
			"gfwlist.txt",
		}
		for _, sp := range searchPaths {
			if data, err := os.ReadFile(sp); err == nil {
				os.WriteFile(gfwDest, data, 0644)
				break
			}
		}
	}

	if err := p.loadConfig(); err == nil {
		log.Printf("[*] Loaded config from %s", *configPath)
		if p.Config.HTTPProxyIface == "" {
			p.Config.HTTPProxyIface = "en0"
		}
	} else {
		log.Printf("Failed to load config: %v", err)
	}

	if err := p.startHTTPProxy(); err != nil {
		log.Fatalf("Failed to start HTTP proxy: %v", err)
	}

	go func() {
		if _, err := p.updateGFWListOnStartup(); err != nil {
			p.addLog(fmt.Sprintf("Failed to update GFWList on startup: %v", err))
		}
	}()

	if p.Config.AutoStart {
		go p.Start()
	}

	setupAPIHandlers(p)

	// Start GUI server
	guiListener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *guiPort))
	if err != nil {
		log.Fatalf("Failed to start GUI server: %v", err)
	}
	*guiPort = guiListener.Addr().(*net.TCPAddr).Port
	fmt.Printf("[*] GUI Console: http://127.0.0.1:%d\n", *guiPort)

	go func() {
		if err := http.Serve(guiListener, nil); err != nil {
			log.Printf("GUI server error: %v", err)
		}
	}()

	systray.Run(func() {
		setPlatformTrayIcon()
		systray.SetTooltip("Smart Proxy")

		mStart := systray.AddMenuItem("Start Proxy", "Start the proxy server")
		mRestart := systray.AddMenuItem("Restart Proxy", "Restart the proxy server")
		mStop := systray.AddMenuItem("Stop Proxy", "Stop the proxy server")
		systray.AddSeparator()
		mOpen := systray.AddMenuItem("Open Configuration", "Open the configuration GUI")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit the application")

		updateMenu := func(running bool) {
			if running {
				mStart.Disable()
				mRestart.Enable()
				mStop.Enable()
				systray.SetTooltip("Smart Proxy: Running")
			} else {
				mStart.Enable()
				mRestart.Disable()
				mStop.Disable()
				systray.SetTooltip("Smart Proxy: Stopped")
			}
		}

		p.onStatusChange = updateMenu
		updateMenu(p.IsRunning())

		go func() {
			for {
				select {
				case <-mStart.ClickedCh:
					if err := p.Start(); err != nil {
						log.Printf("Error starting proxy: %v", err)
					}
				case <-mRestart.ClickedCh:
					if err := p.Restart(); err != nil {
						log.Printf("Error restarting proxy: %v", err)
					}
				case <-mStop.ClickedCh:
					p.Stop()
				case <-mOpen.ClickedCh:
					openBrowser(fmt.Sprintf("http://127.0.0.1:%d", *guiPort))
				case <-mQuit.ClickedCh:
					systray.Quit()
				}
			}
		}()
	}, func() {
		p.Stop()
		releaseLock()
	})
}

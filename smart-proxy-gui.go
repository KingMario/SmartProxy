package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/getlantern/systray"
)

// Socket options for macOS
const (
	IP_BOUND_IF   = 0x19
	IPV6_BOUND_IF = 0x7D
)

// Config represents the proxy configuration
type Config struct {
	Port            int      `json:"port"`
	DefaultIface    string   `json:"defaultIface"`
	GFWIface        string   `json:"gfwIface"`
	CompanyIface    string   `json:"companyIface"`
	GFWListURL      string   `json:"gfwlistUrl"`
	CompanyDomains  []string `json:"companyDomains"`
	ExtraGFWDomains []string `json:"extraGfwDomains"`
	AutoStart       bool     `json:"autoStart"`
}

type ProxyServer struct {
	Config       Config
	GFWDomains   map[string]bool
	IfaceIndices map[string]int
	IfaceIPs     map[string]string
	listener     net.Listener
	running      bool
	mu           sync.RWMutex
	logBuffer    []string
	logMu        sync.Mutex
	configPath   string
}

func (p *ProxyServer) addLog(msg string) {
	p.logMu.Lock()
	defer p.logMu.Unlock()
	ts := time.Now().Format("15:04:05")
	p.logBuffer = append(p.logBuffer, fmt.Sprintf("[%s] %s", ts, msg))
	if len(p.logBuffer) > 100 {
		p.logBuffer = p.logBuffer[1:]
	}
	log.Println(msg)
}

func (p *ProxyServer) saveConfig() error {
	data, err := json.MarshalIndent(p.Config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.configPath, data, 0644)
}

func (p *ProxyServer) loadConfig() error {
	data, err := os.ReadFile(p.configPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &p.Config)
}

func getInterfaceInfo(ifaceName string) (int, string, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return 0, "", err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return iface.Index, "", nil
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return iface.Index, ipnet.IP.String(), nil
			}
		}
	}
	return iface.Index, "", nil
}

func (p *ProxyServer) loadGFWList() error {
	p.mu.RLock()
	url := p.Config.GFWListURL
	p.mu.RUnlock()

	var raw []byte
	var err error

	if strings.HasPrefix(url, "@") || (!strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://")) {
		path := strings.TrimPrefix(url, "@")
		if !filepath.IsAbs(path) {
			path = filepath.Join(filepath.Dir(p.configPath), path)
		}
		raw, err = os.ReadFile(path)
		if err != nil {
			return err
		}
	} else {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		raw, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(string(raw))
	content := ""
	if err == nil {
		content = string(decoded)
	} else {
		content = string(raw)
	}

	p.mu.Lock()
	p.GFWDomains = make(map[string]bool)
	domainRegex := regexp.MustCompile(`([A-Za-z0-9.-]+\.[A-Za-z]{2,})$`)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") || strings.HasPrefix(line, "@@") {
			continue
		}
		line = strings.TrimLeft(line, "|.")
		for _, sep := range []string{"/", "^", "*", "?"} {
			if idx := strings.Index(line, sep); idx != -1 {
				line = line[:idx]
			}
		}
		line = strings.Trim(line, ".")
		match := domainRegex.FindStringSubmatch(line)
		if len(match) > 1 {
			p.GFWDomains[strings.ToLower(match[1])] = true
		}
	}
	p.mu.Unlock()

	p.addLog(fmt.Sprintf("Loaded %d domains from GFWList", len(p.GFWDomains)))
	return nil
}

func (p *ProxyServer) isGFWDomain(host string) bool {
	host = strings.ToLower(host)
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.GFWDomains[host] {
		return true
	}
	for _, domain := range p.Config.ExtraGFWDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	parts := strings.Split(host, ".")
	for i := 0; i < len(parts)-1; i++ {
		suffix := strings.Join(parts[i:], ".")
		if p.GFWDomains[suffix] {
			return true
		}
	}
	return false
}

func (p *ProxyServer) isCompanyDomain(host string) bool {
	host = strings.ToLower(host)
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, domain := range p.Config.CompanyDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func (p *ProxyServer) selectIface(host string) string {
	if net.ParseIP(host) != nil {
		return p.Config.DefaultIface
	}
	if p.isCompanyDomain(host) && p.Config.CompanyIface != "" {
		return p.Config.CompanyIface
	}
	if p.isGFWDomain(host) {
		return p.Config.GFWIface
	}
	return p.Config.DefaultIface
}

func (p *ProxyServer) Start() error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	p.IfaceIndices = make(map[string]int)
	p.IfaceIPs = make(map[string]string)
	for _, name := range []string{p.Config.DefaultIface, p.Config.GFWIface, p.Config.CompanyIface} {
		if name == "" {
			continue
		}
		idx, ip, err := getInterfaceInfo(name)
		if err == nil {
			p.IfaceIndices[name] = idx
			p.IfaceIPs[name] = ip
		}
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p.Config.Port))
	if err != nil {
		p.mu.Unlock()
		return err
	}
	p.listener = ln
	p.running = true
	p.mu.Unlock()

	p.loadGFWList()
	p.addLog(fmt.Sprintf("SOCKS5 Proxy started on 127.0.0.1:%d", p.Config.Port))

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go p.handleConnection(conn)
		}
	}()
	return nil
}

func (p *ProxyServer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.listener != nil {
		p.listener.Close()
		p.listener = nil
	}
	p.running = false
	p.addLog("Proxy server stopped")
}

func (p *ProxyServer) handleConnection(client net.Conn) {
	defer client.Close()
	buf := make([]byte, 256)
	if _, err := io.ReadFull(client, buf[:2]); err != nil || buf[0] != 0x05 {
		return
	}
	nmethods := int(buf[1])
	if _, err := io.ReadFull(client, buf[:nmethods]); err != nil {
		return
	}
	client.Write([]byte{0x05, 0x00})
	if _, err := io.ReadFull(client, buf[:4]); err != nil || buf[0] != 0x05 {
		return
	}
	var host string
	switch buf[3] {
	case 0x01:
		if _, err := io.ReadFull(client, buf[:4]); err != nil {
			return
		}
		host = net.IP(buf[:4]).String()
	case 0x03:
		if _, err := io.ReadFull(client, buf[:1]); err != nil {
			return
		}
		l := int(buf[0])
		if _, err := io.ReadFull(client, buf[:l]); err != nil {
			return
		}
		host = string(buf[:l])
	default:
		return
	}
	if _, err := io.ReadFull(client, buf[:2]); err != nil {
		return
	}
	port := int(buf[0])<<8 | int(buf[1])
	targetAddr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	targetIface := p.selectIface(host)
	p.mu.RLock()
	ifIndex := p.IfaceIndices[targetIface]
	localIP := p.IfaceIPs[targetIface]
	p.mu.RUnlock()

	dialer := &net.Dialer{
		LocalAddr: &net.TCPAddr{IP: net.ParseIP(localIP)},
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				if strings.Contains(network, "tcp6") {
					syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, IPV6_BOUND_IF, ifIndex)
				} else {
					syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, IP_BOUND_IF, ifIndex)
				}
			})
		},
	}

	remote, err := dialer.Dial("tcp", targetAddr)
	if err != nil {
		client.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remote.Close()
	client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(remote, client)
		if tcp, ok := remote.(*net.TCPConn); ok {
			tcp.CloseWrite()
		}
	}()
	go func() {
		defer wg.Done()
		io.Copy(client, remote)
		if tcp, ok := client.(*net.TCPConn); ok {
			tcp.CloseWrite()
		}
	}()
	wg.Wait()
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Println(err)
	}
}

func main() {
	guiPort := flag.Int("gui-port", 10086, "Port for GUI console")
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".smart-proxy")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		os.MkdirAll(configDir, 0755)
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
			Port:         1080,
			DefaultIface: "en0",
			GFWListURL:   filepath.Join(configDir, "gfwlist.txt"),
			AutoStart:    true,
		},
	}

	// Ensure gfwlist.txt exists in configDir
	gfwDest := filepath.Join(configDir, "gfwlist.txt")
	if _, err := os.Stat(gfwDest); os.IsNotExist(err) {
		// Try to find it in the same directory as the executable or Resources
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
		if p.Config.AutoStart {
			go p.Start()
		}
	}

	http.HandleFunc("/api/interfaces", func(w http.ResponseWriter, r *http.Request) {
		ifaces, _ := net.Interfaces()
		var list []map[string]interface{}
		for _, iface := range ifaces {
			addrs, _ := iface.Addrs()
			if len(addrs) > 0 {
				list = append(list, map[string]interface{}{"name": iface.Name, "index": iface.Index})
			}
		}
		json.NewEncoder(w).Encode(list)
	})

	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var cfg Config
			json.NewDecoder(r.Body).Decode(&cfg)
			p.mu.Lock()
			p.Config = cfg
			p.mu.Unlock()
			p.saveConfig()
			w.WriteHeader(http.StatusOK)
			return
		}
		p.mu.RLock()
		json.NewEncoder(w).Encode(p.Config)
		p.mu.RUnlock()
	})

	http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		p.mu.RLock()
		p.logMu.Lock()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"running": p.running,
			"port":    p.Config.Port,
			"logs":    p.logBuffer,
		})
		p.logMu.Unlock()
		p.mu.RUnlock()
	})

	http.HandleFunc("/api/start", func(w http.ResponseWriter, r *http.Request) {
		if err := p.Start(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/api/stop", func(w http.ResponseWriter, r *http.Request) {
		p.Stop()
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Smart Proxy Control Panel</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        body { background: #f8f9fa; padding: 20px; font-family: sans-serif; }
        .card { margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.05); }
        #log { background: #1e1e1e; color: #00ff00; height: 500px; overflow-y: scroll; font-family: monospace; padding: 10px; font-size: 12px; }
        .status-on { color: #28a745; font-weight: bold; }
        .status-off { color: #dc3545; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container-fluid px-4">
        <div class="d-flex justify-content-between align-items-center my-4">
            <h2>🚀 Smart Proxy</h2>
            <div id="statusBadge"></div>
        </div>

        <div class="row align-items-start">
            <div class="col-lg-5 col-md-12">
                <div class="card">
                    <div class="card-header fw-bold">General Settings</div>
                    <div class="card-body">
                        <div class="mb-3"><label class="form-label">SOCKS5 Port</label><input type="number" id="proxyPort" class="form-control"></div>
                        <div class="mb-3"><label class="form-label">Default Interface</label><select id="defaultIface" class="form-select"></select></div>
                        <div class="mb-3"><label class="form-label">GFW Interface (Personal VPN)</label><select id="gfwIface" class="form-select"></select></div>
                        <div class="mb-3"><label class="form-label">Company Interface (Company VPN)</label><select id="companyIface" class="form-select"></select></div>
                    </div>
                </div>
                
                <div class="card">
                    <div class="card-header fw-bold">Rules & Settings</div>
                    <div class="card-body">
                        <div class="mb-3"><label class="form-label">Company Domains</label><textarea id="companyDomains" class="form-control" rows="2" placeholder="e.g. company.com, internal.net"></textarea></div>
                        <div class="mb-3"><label class="form-label">Extra GFW Domains</label><textarea id="extraGfwDomains" class="form-control" rows="2" placeholder="e.g. gvt2.com, google.com"></textarea></div>
                        <div class="mb-3"><label class="form-label">GFWList URL/Path</label><input id="gfwlistUrl" class="form-control"></div>
                        <div class="form-check form-switch mt-3">
                            <input class="form-check-input" type="checkbox" id="autoStart">
                            <label class="form-check-label" for="autoStart">Auto-start proxy on program launch</label>
                        </div>
                    </div>
                </div>
                
                <div class="d-grid gap-2 mb-4">
                    <button id="btnStart" class="btn btn-primary btn-lg" onclick="control('start')">Start Proxy</button>
                    <button id="btnStop" class="btn btn-danger btn-lg" onclick="control('stop')">Stop Proxy</button>
                    <button class="btn btn-outline-secondary" onclick="saveConfig()">Save Configuration</button>
                </div>
            </div>

            <div class="col-lg-7 col-md-12">
                <div class="card">
                    <div class="card-header fw-bold d-flex justify-content-between align-items-center">
                        Real-time Logs
                        <button class="btn btn-sm btn-outline-danger" onclick="document.getElementById('log').innerHTML=''">Clear</button>
                    </div>
                    <div class="card-body p-0"><div id="log"></div></div>
                </div>
            </div>
        </div>
    </div>

    <script>
        async function loadData() {
            try {
                const ifaces = await fetch('/api/interfaces').then(r => r.json());
                ['defaultIface', 'gfwIface', 'companyIface'].forEach(id => {
                    const sel = document.getElementById(id);
                    const currentVal = sel.value;
                    sel.innerHTML = '<option value="">None</option>' + ifaces.map(i => `+"`"+`<option value="${i.name}">${i.name}</option>`+"`"+`).join('');
                    if(currentVal) sel.value = currentVal;
                });

                const config = await fetch('/api/config').then(r => r.json());
                document.getElementById('proxyPort').value = config.port || 1080;
                document.getElementById('defaultIface').value = config.defaultIface || '';
                document.getElementById('gfwIface').value = config.gfwIface || '';
                document.getElementById('companyIface').value = config.companyIface || '';
                document.getElementById('companyDomains').value = (config.companyDomains || []).join(', ');
                document.getElementById('extraGfwDomains').value = (config.extraGfwDomains || []).join(', ');
                document.getElementById('gfwlistUrl').value = config.gfwlistUrl || '';
                document.getElementById('autoStart').checked = config.autoStart;
            } catch(e) { console.error("load error", e); }
        }

        async function saveConfig() {
            const body = {
                port: parseInt(document.getElementById('proxyPort').value),
                defaultIface: document.getElementById('defaultIface').value,
                gfwIface: document.getElementById('gfwIface').value,
                companyIface: document.getElementById('companyIface').value,
                companyDomains: document.getElementById('companyDomains').value.split(',').map(s => s.trim()).filter(s => s),
                extraGfwDomains: document.getElementById('extraGfwDomains').value.split(',').map(s => s.trim()).filter(s => s),
                gfwlistUrl: document.getElementById('gfwlistUrl').value,
                autoStart: document.getElementById('autoStart').checked
            };
            await fetch('/api/config', { method: 'POST', body: JSON.stringify(body) });
            alert('Configuration saved');
        }

        async function control(action) {
            await fetch('/api/' + action, { method: 'POST' });
            updateStatus();
        }

        let lastLogCount = 0;
        async function updateStatus() {
            try {
                const status = await fetch('/api/status').then(r => r.json());
                const port = status.port || 1080;
                document.getElementById('statusBadge').innerHTML = status.running ? `+"`"+`<span class="status-on">● Running (127.0.0.1:${port})</span>`+"`"+` : '<span class="status-off">○ Stopped</span>';
                document.getElementById('btnStart').disabled = status.running;
                document.getElementById('btnStop').disabled = !status.running;
                
                const logDiv = document.getElementById('log');
                if (status.logs && status.logs.length !== lastLogCount) {
                    logDiv.innerHTML = status.logs.join('<br>');
                    logDiv.scrollTop = logDiv.scrollHeight;
                    lastLogCount = status.logs.length;
                }
            } catch(e) {}
        }

        loadData();
        setInterval(updateStatus, 1000);
    </script>
</body>
</html>
		`)
	})

	go func() {
		fmt.Printf("[*] GUI Console: http://127.0.0.1:%d\n", *guiPort)
		if err := http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", *guiPort), nil); err != nil {
			log.Printf("GUI server error: %v", err)
		}
	}()

	systray.Run(func() {
		systray.SetTitle("🚀")
		systray.SetTooltip("Smart Proxy")
		mOpen := systray.AddMenuItem("Open Configuration", "Open the configuration GUI")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("Quit", "Quit the application")

		go func() {
			for {
				select {
				case <-mOpen.ClickedCh:
					openBrowser(fmt.Sprintf("http://127.0.0.1:%d", *guiPort))
				case <-mQuit.ClickedCh:
					systray.Quit()
				}
			}
		}()
	}, func() {
		p.Stop()
	})
}

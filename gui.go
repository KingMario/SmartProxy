package main

const controlPanelHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Smart Proxy Control Panel</title>
    <link href="/assets/ui/bootstrap.min.css" rel="stylesheet">
    <link href="/assets/ui/bootstrap-icons.css" rel="stylesheet">
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
            <div class="d-flex align-items-center gap-3">
                <h2 class="mb-0">🚀 Smart Proxy</h2>
                <div class="btn-group">
                    <button id="btnStart" class="btn btn-sm btn-outline-primary" onclick="control('start')" title="Start"><i class="bi bi-play-fill"></i></button>
                    <button id="btnRestart" class="btn btn-sm btn-outline-warning" onclick="control('restart')" title="Restart"><i class="bi bi-arrow-clockwise"></i></button>
                    <button id="btnStop" class="btn btn-sm btn-outline-danger" onclick="control('stop')" title="Stop"><i class="bi bi-stop-fill"></i></button>
                    <button class="btn btn-sm btn-outline-secondary" onclick="saveConfig()" title="Save"><i class="bi bi-save"></i></button>
                </div>
            </div>
            <div id="statusBadge"></div>
        </div>

        <div class="row align-items-start">
            <div class="col-lg-5 col-md-12">
                <div class="card">
                    <div class="card-header fw-bold">Quick Settings</div>
                    <div class="card-body">
                        <div class="form-check form-switch">
                            <input class="form-check-input" type="checkbox" id="autoStart">
                            <label class="form-check-label" for="autoStart">Auto-start proxy on program launch</label>
                        </div>
                        <div class="form-check form-switch mt-2">
                            <input class="form-check-input" type="checkbox" id="verboseLog">
                            <label class="form-check-label" for="verboseLog">Verbose Logging (Show routing decisions)</label>
                        </div>
                    </div>
                </div>

                <div class="card">
                    <div class="card-header fw-bold">General Settings</div>
                    <div class="card-body">
                        <div class="mb-3"><label class="form-label">SOCKS5 Port</label><input type="number" id="proxyPort" class="form-control"></div>
                        <div class="mb-3"><label class="form-label">Default Interface</label><select id="defaultIface" class="form-select"></select></div>
                        <div class="mb-3">
                            <label class="form-label">GFW Interface (Personal VPN)</label>
                            <div class="input-group">
                                <select id="gfwIface" class="form-select"></select>
                                <button class="btn btn-outline-secondary" type="button" onclick="autoDetectGFW()" title="Auto Detect"><i class="bi bi-search"></i> Detect</button>
                            </div>
                        </div>
                        <div class="mb-3">
                            <label class="form-label" id="gfwProxyLabel">GFW Upstream Proxy (e.g. 127.0.0.1:8787)</label>
                            <input type="text" id="gfwProxy" class="form-control" placeholder="Optional, for Lantern etc." oninput="updateGfwStatus()">
                        </div>
                        <div class="mb-3">
                            <label class="form-label">Company Interface (Company VPN)</label>
                            <div class="input-group">
                                <select id="companyIface" class="form-select"></select>
                                <button class="btn btn-outline-secondary" type="button" onclick="autoDetectCompany()" title="Auto Detect"><i class="bi bi-search"></i> Detect</button>
                            </div>
                        </div>
                    </div>
                </div>
                
                    <div class="card">
                        <div class="card-header fw-bold">HTTP Proxy</div>
                        <div class="card-body">
                            <div class="mb-3">
                                <label class="form-label">Outgoing Interface</label>
                                <select id="httpProxyIface" class="form-select"></select>
                            </div>
                            <div class="alert alert-light mb-0 py-2">
                                HTTP proxy port: <strong><span id="httpProxyStatus">Starting...</span></strong>
                            </div>
                        </div>
                    </div>
                    
                    <div class="card">
                        <div class="card-header fw-bold">Rules & Settings</div>
                        <div class="card-body">
                        <div class="mb-3"><label class="form-label">Company Domains</label><textarea id="companyDomains" class="form-control" rows="2" placeholder="e.g. company.com, internal.net"></textarea></div>
                        <div class="mb-3"><label class="form-label">Bypass Domains (Direct)</label><textarea id="bypassDomains" class="form-control" rows="2" placeholder="e.g. example.com, local.dev"></textarea></div>
                        <div class="mb-3"><label class="form-label">Extra GFW Domains</label><textarea id="extraGfwDomains" class="form-control" rows="2" placeholder="e.g. gvt2.com, google.com"></textarea></div>
                        <div class="mb-3"><label class="form-label">GFWList URL/Path</label><input id="gfwlistUrl" class="form-control"></div>
                    </div>
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

        <div class="position-fixed bottom-0 end-0 p-3" style="z-index: 11">
            <div id="liveToast" class="toast align-items-center text-white bg-success border-0" role="alert" aria-live="assertive" aria-atomic="true">
                <div class="d-flex">
                    <div class="toast-body">
                        Configuration saved successfully!
                    </div>
                    <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast" aria-label="Close"></button>
                </div>
            </div>
        </div>
    </div>

    <script src="/assets/ui/bootstrap.bundle.min.js"></script>
    <script>
        async function refreshInterfaces() {
            const ifaces = await fetch('/api/interfaces').then(r => r.json());
            ['defaultIface', 'gfwIface', 'companyIface', 'httpProxyIface'].forEach(id => {
                const sel = document.getElementById(id);
                const currentVal = sel.value;
                sel.innerHTML = '<option value="">None</option>' + ifaces.map(i => `+"`"+`<option value="${i.name}">${i.name}</option>`+"`"+`).join('');
                if(currentVal) sel.value = currentVal;
            });
        }

        async function loadData() {
            try {
                await refreshInterfaces();
                const config = await fetch('/api/config').then(r => r.json());
                document.getElementById('proxyPort').value = config.port || 1080;
                document.getElementById('defaultIface').value = config.defaultIface || '';
                document.getElementById('gfwIface').value = config.gfwIface || '';
                document.getElementById('gfwProxy').value = config.gfwProxy || '';
                try {
                    const sysProxy = await fetch('/api/detect-system-proxy').then(r => r.json());
                    if (sysProxy.proxy) {
                        document.getElementById('gfwProxyLabel').innerText = 'GFW Upstream Proxy (e.g. ' + sysProxy.proxy + ')';
                    }
                } catch(e) {}
                document.getElementById('companyIface').value = config.companyIface || '';
                document.getElementById('httpProxyIface').value = config.httpProxyIface || '';
                document.getElementById('companyDomains').value = (config.companyDomains || []).join(', ');
                document.getElementById('bypassDomains').value = (config.bypassDomains || []).join(', ');
                document.getElementById('extraGfwDomains').value = (config.extraGfwDomains || []).join(', ');
                document.getElementById('gfwlistUrl').value = config.gfwlistUrl || '';
                document.getElementById('autoStart').checked = config.autoStart;
                document.getElementById('verboseLog').checked = config.verboseLog;
                updateGfwStatus();
            } catch(e) { console.error("load error", e); }
        }

        function showToast(message) {
            document.querySelector('#liveToast .toast-body').innerText = message;
            const toast = new bootstrap.Toast(document.getElementById('liveToast'));
            toast.show();
        }

        async function saveConfig() {
            const body = {
                port: parseInt(document.getElementById('proxyPort').value),
                defaultIface: document.getElementById('defaultIface').value,
                gfwIface: document.getElementById('gfwIface').value,
                gfwProxy: document.getElementById('gfwProxy').value,
                companyIface: document.getElementById('companyIface').value,
                companyDomains: document.getElementById('companyDomains').value.split(',').map(s => s.trim()).filter(s => s),
                bypassDomains: document.getElementById('bypassDomains').value.split(',').map(s => s.trim()).filter(s => s),
                extraGfwDomains: document.getElementById('extraGfwDomains').value.split(',').map(s => s.trim()).filter(s => s),
                gfwlistUrl: document.getElementById('gfwlistUrl').value,
                httpProxyIface: document.getElementById('httpProxyIface').value,
                autoStart: document.getElementById('autoStart').checked,
                verboseLog: document.getElementById('verboseLog').checked
            };
            await fetch('/api/config', { method: 'POST', body: JSON.stringify(body) });
			showToast('Configuration saved successfully!');
            updateGfwStatus();
        }

        function updateGfwStatus() {
            const proxy = document.getElementById('gfwProxy').value.trim();
            const iface = document.getElementById('gfwIface');
            const btn = document.querySelector('button[onclick="autoDetectGFW()"]');
            const label = document.querySelector('label[for="gfwIface"]') || iface.parentNode.previousElementSibling;
            
            if (proxy) {
                iface.disabled = true;
                btn.disabled = true;
                if (!document.getElementById('gfwOverrideHint')) {
                    const hint = document.createElement('small');
                    hint.id = 'gfwOverrideHint';
                    hint.className = 'text-danger d-block mt-1';
                    hint.style.fontSize = '11px';
                    hint.innerText = '⚠️ Overridden by Upstream Proxy';
                    iface.parentNode.after(hint);
                }
            } else {
                iface.disabled = false;
                btn.disabled = false;
                const hint = document.getElementById('gfwOverrideHint');
                if (hint) hint.remove();
            }
        }

        async function control(action) {
            await fetch('/api/' + action, { method: 'POST' });
            updateStatus();
        }

        async function autoDetectGFW() {
            const btn = event.target.closest('button');
            const originalHtml = btn.innerHTML;
            btn.disabled = true;
            btn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> Testing...';
            try {
                await refreshInterfaces();
                const res = await fetch('/api/autodetect-gfw', { method: 'POST' }).then(r => r.json());
                document.getElementById('gfwIface').value = res.iface || '';
                showToast(res.iface ? `+"`"+`Auto-detected GFW Interface: ${res.iface}`+"`"+` : 'No working GFW interface found.');
            } finally {
                btn.disabled = false;
                btn.innerHTML = originalHtml;
            }
        }

        async function autoDetectCompany() {
            const btn = event.target.closest('button');
            const originalHtml = btn.innerHTML;
            btn.disabled = true;
            btn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> Testing...';
            try {
                await refreshInterfaces();
                const res = await fetch('/api/autodetect-company', { method: 'POST' }).then(r => r.json());
                document.getElementById('companyIface').value = res.iface || '';
                showToast(res.iface ? `+"`"+`Auto-detected Company Interface: ${res.iface}`+"`"+` : 'No working Company interface found.');
            } finally {
                btn.disabled = false;
                btn.innerHTML = originalHtml;
            }
        }

        let lastLogContent = "";
        async function updateStatus() {
            try {
                const status = await fetch('/api/status').then(r => r.json());
                const port = status.port || 1080;
                const httpStatus = document.getElementById('httpProxyStatus');
                if (httpStatus) {
                    const httpPort = status.httpProxyPort;
                    const httpIface = status.httpProxyIface || 'HTTP';
                    httpStatus.innerText = httpPort ? httpIface + ' -> 0.0.0.0:' + httpPort : 'Starting...';
                }
                document.getElementById('statusBadge').innerHTML = status.running ? `+"`"+`<span class="status-on">● Running (0.0.0.0:${port})</span>`+"`"+` : '<span class="status-off">○ Stopped</span>';
                document.getElementById('btnStart').disabled = status.running;
                document.getElementById('btnRestart').disabled = !status.running;
                document.getElementById('btnStop').disabled = !status.running;
                
                const logDiv = document.getElementById('log');
                const currentLogs = (status.logs || []).join('<br>');
                if (currentLogs !== lastLogContent) {
                    logDiv.innerHTML = currentLogs;
                    logDiv.scrollTop = logDiv.scrollHeight;
                    lastLogContent = currentLogs;
                }
            } catch(e) {}
        }

        document.addEventListener('keydown', function(e) {
			const key = e.key.toLowerCase();
			if ((e.metaKey || e.ctrlKey) && key === 's') {
                e.preventDefault();
                saveConfig();
				return;
			}
			if ((e.metaKey || e.ctrlKey) && key === 'r') {
				e.preventDefault();
				control('restart');
            }
        });

        loadData();
        setInterval(updateStatus, 1000);
    </script>
</body>
</html>
`

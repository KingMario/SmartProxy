import Cocoa
import Foundation

class AppDelegate: NSObject, NSApplicationDelegate {
    var statusItem: NSStatusItem?
    var proxyProcess: Process?
    let guiURL = "http://127.0.0.1:10086"
    
    func applicationDidFinishLaunching(_ notification: Notification) {
        // 设置为后台应用（不在 Dock 显示，仅在 Menu Bar）
        NSApp.setActivationPolicy(.accessory)
        
        // 创建 Menu Bar Item
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        
        if let button = statusItem?.button {
            button.title = "🚀"
        }
        
        setupMenu()
        startProxy()
    }
    
    func setupMenu() {
        let menu = NSMenu()
        menu.addItem(NSMenuItem(title: "Open GUI", action: #selector(openGUI), keyEquivalent: "g"))
        menu.addItem(NSMenuItem.separator())
        menu.addItem(NSMenuItem(title: "Quit", action: #selector(quitApp), keyEquivalent: "q"))
        statusItem?.menu = menu
    }
    
    @objc func openGUI() {
        if let url = URL(string: guiURL) {
            NSWorkspace.shared.open(url)
        }
    }
    
    func startProxy() {
        let task = Process()
        guard let bundlePath = Bundle.main.resourcePath else { return }
        let executablePath = (bundlePath as NSString).appendingPathComponent("smart-proxy-gui")
        
        task.executableURL = URL(fileURLWithPath: executablePath)
        
        // 环境变量和工作目录
        let home = FileManager.default.homeDirectoryForCurrentUser
        let configDir = home.appendingPathComponent(".smart-proxy")
        
        // 确保目录存在
        try? FileManager.default.createDirectory(at: configDir, withIntermediateDirectories: true)
        
        task.currentDirectoryURL = configDir
        
        // 日志处理
        let logPath = configDir.appendingPathComponent("output.log").path
        if let logHandle = FileHandle(forWritingAtPath: logPath) ?? (FileManager.default.createFile(atPath: logPath, contents: nil) ? FileHandle(forWritingAtPath: logPath) : nil) {
            logHandle.seekToEndOfFile()
            task.standardOutput = logHandle
            task.standardError = logHandle
        }
        
        do {
            try task.run()
            self.proxyProcess = task
        } catch {
            let alert = NSAlert()
            alert.messageText = "Error"
            alert.informativeText = "Failed to start smart-proxy-gui: \(error.localizedDescription)"
            alert.runModal()
        }
    }
    
    @objc func quitApp() {
        proxyProcess?.terminate()
        NSApplication.shared.terminate(nil)
    }
    
    func applicationWillTerminate(_ notification: Notification) {
        proxyProcess?.terminate()
    }
}

let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.run()

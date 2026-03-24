//go:build !windows

package llm

import (
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// daemonSysProcAttr gibt SysProcAttr zurück das den Prozess vom Parent löst.
func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

// Stop beendet den Daemon-Server anhand der PID-Datei.
func (m *Manager) Stop() {
	pidFile := filepath.Join(m.cfgDir, "batch-cost-llm.pid")
	if data, err := os.ReadFile(pidFile); err == nil {
		if pid, err := strconv.Atoi(string(data)); err == nil {
			if proc, err := os.FindProcess(pid); err == nil {
				_ = proc.Kill()
			}
		}
	}
	_ = os.Remove(pidFile)
	if m.proc != nil && m.proc.Process != nil {
		_ = m.proc.Process.Kill()
		_ = m.proc.Wait()
	}
}

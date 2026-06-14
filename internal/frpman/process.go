package frpman

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// ProcState is the externally visible state of the frp subprocess.
type ProcState struct {
	Running   bool   `json:"running"`
	PID       int    `json:"pid,omitempty"`
	StartedAt string `json:"startedAt,omitempty"`
	UptimeSec int64  `json:"uptimeSec"`
	LastError string `json:"lastError,omitempty"`
	ExitCode  *int   `json:"exitCode,omitempty"`
	BinPath   string `json:"binPath,omitempty"`
}

// ProcessManager runs a single frp binary (frps or frpc) and tracks its state.
type ProcessManager struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	startedAt time.Time
	running   bool
	stopping  bool
	lastError string
	exitCode  *int
	done      chan struct{}

	// retained for restart
	binPath string
	args    []string
	workDir string

	logs *LogHub
}

// NewProcessManager returns a manager writing process output to hub.
func NewProcessManager(hub *LogHub) *ProcessManager {
	return &ProcessManager{logs: hub}
}

// Logs exposes the underlying log hub.
func (pm *ProcessManager) Logs() *LogHub { return pm.logs }

// Start launches binPath with args in workDir. It is an error to start when a
// process is already running.
func (pm *ProcessManager) Start(binPath string, args []string, workDir string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.running {
		return errors.New("frp 已在运行")
	}
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("找不到 frp 可执行文件: %w", err)
	}

	cmd := exec.Command(binPath, args...)
	cmd.Dir = workDir
	// Discourage colored output; any residual ANSI codes are stripped downstream.
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "CLICOLOR=0")
	lw := &lineWriter{hub: pm.logs}
	cmd.Stdout = lw
	cmd.Stderr = lw

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 frp 失败: %w", err)
	}

	pm.cmd = cmd
	pm.running = true
	pm.stopping = false
	pm.lastError = ""
	pm.exitCode = nil
	pm.startedAt = time.Now()
	pm.binPath = binPath
	pm.args = args
	pm.workDir = workDir
	pm.done = make(chan struct{})

	pm.logs.Append(fmt.Sprintf("[panel] 已启动 %s (pid %d)", binPath, cmd.Process.Pid))
	go pm.wait(cmd, pm.done)
	return nil
}

func (pm *ProcessManager) wait(cmd *exec.Cmd, done chan struct{}) {
	err := cmd.Wait()

	pm.mu.Lock()
	pm.running = false
	code := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	}
	pm.exitCode = &code
	wasStopping := pm.stopping
	if err != nil && !wasStopping {
		pm.lastError = err.Error()
		pm.logs.Append(fmt.Sprintf("[panel] frp 进程异常退出: %v", err))
	} else {
		pm.logs.Append("[panel] frp 进程已停止")
	}
	pm.mu.Unlock()

	close(done)
}

// Stop terminates the process gracefully, escalating to kill after a timeout.
func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()
	if !pm.running || pm.cmd == nil || pm.cmd.Process == nil {
		pm.mu.Unlock()
		return nil
	}
	pm.stopping = true
	proc := pm.cmd.Process
	done := pm.done
	pm.mu.Unlock()

	if runtime.GOOS == "windows" {
		// Windows has no SIGTERM delivery for console apps via os/exec; kill.
		_ = proc.Kill()
	} else {
		_ = proc.Signal(syscall.SIGTERM)
		select {
		case <-done:
			return nil
		case <-time.After(5 * time.Second):
			_ = proc.Kill()
		}
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		return errors.New("停止超时")
	}
	return nil
}

// Restart stops then starts with the previously used parameters.
func (pm *ProcessManager) Restart() error {
	pm.mu.Lock()
	bin, args, dir := pm.binPath, pm.args, pm.workDir
	pm.mu.Unlock()
	if bin == "" {
		return errors.New("尚未启动过 frp")
	}
	if err := pm.Stop(); err != nil {
		return err
	}
	return pm.Start(bin, args, dir)
}

// Running reports whether the process is currently up.
func (pm *ProcessManager) Running() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.running
}

// State returns a snapshot of the current process state.
func (pm *ProcessManager) State() ProcState {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	st := ProcState{
		Running:   pm.running,
		LastError: pm.lastError,
		ExitCode:  pm.exitCode,
		BinPath:   pm.binPath,
	}
	if pm.running && pm.cmd != nil && pm.cmd.Process != nil {
		st.PID = pm.cmd.Process.Pid
		st.StartedAt = pm.startedAt.Format(time.RFC3339)
		st.UptimeSec = int64(time.Since(pm.startedAt).Seconds())
	}
	return st
}

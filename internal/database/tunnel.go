package database

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	execLookPath = exec.LookPath
)

// SSHConfig describes an optional SSH tunnel used to reach the database.
type SSHConfig struct {
	Host      string
	Port      int
	User      string
	KeyFile   string
	LocalPort int
}

// Enabled reports whether SSH tunneling should be used.
func (c SSHConfig) Enabled() bool {
	return strings.TrimSpace(c.Host) != ""
}

// StartSSHTunnel starts an SSH local-forward tunnel and mutates the connection
// to use the forwarded localhost endpoint.
func StartSSHTunnel(ctx context.Context, conn *Connection) (func() error, error) {
	if conn == nil || !conn.SSH.Enabled() {
		return nil, nil
	}

	if _, err := execLookPath("ssh"); err != nil {
		return nil, fmt.Errorf("ssh client is required but not found in PATH: %w", err)
	}

	localPort, err := resolveTunnelLocalPort(conn.SSH.LocalPort)
	if err != nil {
		return nil, err
	}

	target := conn.SSH.Host
	if conn.SSH.User != "" {
		target = conn.SSH.User + "@" + target
	}

	args := []string{
		"-N",
		"-o", "ExitOnForwardFailure=yes",
		"-L", fmt.Sprintf("%d:%s:%d", localPort, conn.Host, conn.Port),
		"-p", strconv.Itoa(conn.SSH.Port),
	}
	if conn.SSH.KeyFile != "" {
		args = append(args, "-i", conn.SSH.KeyFile)
	}
	args = append(args, target)

	stderr := &syncBuffer{}
	cmd := execCommandContext(ctx, "ssh", args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.MultiWriter(os.Stderr, stderr)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ssh tunnel: %w", err)
	}

	tunnel := &tunnelProcess{cmd: cmd, done: make(chan struct{})}
	go func() {
		tunnel.waitErr = cmd.Wait()
		close(tunnel.done)
	}()

	if err := waitForTunnel(localPort, tunnel, stderr.String); err != nil {
		_ = terminateTunnel(tunnel)
		return nil, err
	}

	conn.Host = "127.0.0.1"
	conn.Port = localPort

	return func() error {
		return terminateTunnel(tunnel)
	}, nil
}

// tunnelProcess tracks the ssh child process and the single result of cmd.Wait().
// done is closed once the process exits; waitErr is written before that close, so
// any goroutine observing done as closed can read waitErr safely. A closed channel
// (rather than a one-shot value channel) lets both waitForTunnel and terminateTunnel
// observe the exit without racing to consume the sole result — which previously
// deadlocked terminateTunnel when ssh exited before the tunnel became ready.
type tunnelProcess struct {
	cmd     *exec.Cmd
	done    chan struct{}
	waitErr error
}

func resolveTunnelLocalPort(requested int) (int, error) {
	if requested > 0 {
		return requested, nil
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("failed to allocate local tunnel port: %w", err)
	}
	defer func() { _ = listener.Close() }()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("failed to inspect local tunnel port")
	}

	return addr.Port, nil
}

func waitForTunnel(localPort int, tunnel *tunnelProcess, stderrSnapshot func() string) error {
	deadline := time.Now().Add(5 * time.Second)
	address := fmt.Sprintf("127.0.0.1:%d", localPort)

	for time.Now().Before(deadline) {
		select {
		case <-tunnel.done:
			return fmt.Errorf("ssh tunnel exited before becoming ready: %w: %s", tunnel.waitErr, strings.TrimSpace(stderrSnapshot()))
		default:
		}

		conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}

		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("ssh tunnel did not become ready within 5s: %s", strings.TrimSpace(stderrSnapshot()))
}

func terminateTunnel(tunnel *tunnelProcess) error {
	cmd := tunnel.cmd
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		_ = cmd.Process.Kill()
	}

	select {
	case <-tunnel.done:
		return tunnelExitErr(tunnel.waitErr)
	case <-time.After(2 * time.Second):
		if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return err
		}
		<-tunnel.done
		return tunnelExitErr(tunnel.waitErr)
	}
}

// tunnelExitErr normalises the result of cmd.Wait() for the ssh tunnel, treating a
// process we deliberately terminated (SIGTERM/SIGKILL) as a clean shutdown.
func tunnelExitErr(err error) error {
	if err == nil || errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == -1 {
		return nil
	}
	return err
}

// syncBuffer is a minimal concurrency-safe buffer. os/exec writes a command's
// stderr from an internal goroutine, so the buffer may be read for diagnostics
// while that goroutine is still writing.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

package database

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStartSSHTunnelMutatesConnectionToLocalForward(t *testing.T) {
	originalLookPath := execLookPath
	originalExecCommandContext := execCommandContext
	execLookPath = func(file string) (string, error) { return "/usr/bin/ssh", nil }
	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		helperArgs := append([]string{"-test.run=TestHelperSSHTunnelProcess", "--"}, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], helperArgs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_SSH_TUNNEL=1")
		return cmd
	}
	t.Cleanup(func() {
		execLookPath = originalLookPath
		execCommandContext = originalExecCommandContext
	})

	conn := &Connection{
		Host:     "db.internal",
		Port:     3306,
		User:     "root",
		Password: "secret",
		Database: "testdb",
		SSH: SSHConfig{
			Host:      "bastion.example.com",
			Port:      22,
			User:      "deploy",
			LocalPort: 0,
		},
	}

	stop, err := StartSSHTunnel(context.Background(), conn)
	if err != nil {
		t.Fatalf("StartSSHTunnel returned error: %v", err)
	}
	if stop == nil {
		t.Fatal("expected tunnel cleanup function")
	}
	t.Cleanup(func() {
		if err := stop(); err != nil {
			t.Fatalf("stop returned error: %v", err)
		}
	})

	if conn.Host != "127.0.0.1" {
		t.Fatalf("expected tunneled host 127.0.0.1, got %q", conn.Host)
	}
	if conn.Port == 0 {
		t.Fatal("expected non-zero tunneled port")
	}

	socket, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", conn.Host, conn.Port), 250*time.Millisecond)
	if err != nil {
		t.Fatalf("expected forwarded port to accept connections: %v", err)
	}
	_ = socket.Close()
}

func TestStartSSHTunnelRequiresSSHClient(t *testing.T) {
	originalLookPath := execLookPath
	execLookPath = func(file string) (string, error) { return "", exec.ErrNotFound }
	t.Cleanup(func() {
		execLookPath = originalLookPath
	})

	_, err := StartSSHTunnel(context.Background(), &Connection{
		Host: "db.internal", Port: 3306,
		SSH: SSHConfig{Host: "bastion.example.com", Port: 22, User: "deploy"},
	})
	if err == nil || !strings.Contains(err.Error(), "ssh client is required") {
		t.Fatalf("expected ssh-client error, got %v", err)
	}
}

func TestHelperSSHTunnelProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_SSH_TUNNEL") != "1" {
		return
	}

	args := os.Args
	sep := -1
	for i, arg := range args {
		if arg == "--" {
			sep = i
			break
		}
	}
	if sep == -1 || sep+1 >= len(args) {
		os.Exit(2)
	}

	sshArgs := args[sep+1:]
	var forward string
	for i := 0; i < len(sshArgs); i++ {
		if sshArgs[i] == "-L" && i+1 < len(sshArgs) {
			forward = sshArgs[i+1]
			break
		}
	}
	if forward == "" {
		os.Exit(2)
	}

	parts := strings.SplitN(forward, ":", 3)
	if len(parts) != 3 {
		os.Exit(2)
	}

	port, err := strconv.Atoi(parts[0])
	if err != nil {
		os.Exit(2)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		os.Exit(2)
	}
	defer func() { _ = listener.Close() }()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigCh)

	for {
		if err := listener.(*net.TCPListener).SetDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
			os.Exit(2)
		}

		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
		}

		select {
		case <-sigCh:
			os.Exit(0)
		default:
		}
	}
}

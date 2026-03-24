package clutch

import (
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"

	"github.com/dpopsuev/djinn/driver"
)

func TestSocketTransport_RoundTrip(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "test.sock")

	ln, err := Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// Backend connects in a goroutine (Accept blocks).
	var backend *SocketTransport
	var connectErr error
	done := make(chan struct{})
	go func() {
		backend, connectErr = Connect(sock)
		close(done)
	}()

	shell, err := ln.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer shell.Close()
	<-done
	if connectErr != nil {
		t.Fatal(connectErr)
	}
	defer backend.Close()

	// Shell → Backend
	shell.SendToBackend(ShellMsg{Type: ShellPrompt, Text: "hello"}) //nolint:errcheck
	msg, err := backend.RecvFromShell()
	if err != nil {
		t.Fatal(err)
	}
	if msg.Type != ShellPrompt || msg.Text != "hello" {
		t.Fatalf("got %+v", msg)
	}

	// Backend → Shell
	backend.SendToShell(BackendMsg{Type: BackendText, Text: "world"}) //nolint:errcheck
	resp, err := shell.RecvFromBackend()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Type != BackendText || resp.Text != "world" {
		t.Fatalf("got %+v", resp)
	}
}

func TestSocketTransport_ToolCallWithInput(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "tool.sock")

	ln, err := Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var backend *SocketTransport
	done := make(chan struct{})
	go func() {
		backend, _ = Connect(sock)
		close(done)
	}()
	shell, _ := ln.Accept()
	defer shell.Close()
	<-done
	defer backend.Close()

	// Send a tool call with JSON input through the socket.
	backend.SendToShell(BackendMsg{ //nolint:errcheck
		Type: BackendToolCall,
		ToolCall: &driver.ToolCall{
			ID:    "call-1",
			Name:  "Read",
			Input: json.RawMessage(`{"path":"main.go"}`),
		},
	})

	resp, err := shell.RecvFromBackend()
	if err != nil {
		t.Fatal(err)
	}
	if resp.ToolCall == nil {
		t.Fatal("tool call should not be nil")
	}
	if resp.ToolCall.Name != "Read" {
		t.Fatalf("tool name = %q", resp.ToolCall.Name)
	}
	if string(resp.ToolCall.Input) != `{"path":"main.go"}` {
		t.Fatalf("tool input = %q", string(resp.ToolCall.Input))
	}
}

func TestSocketTransport_Close(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "close.sock")

	ln, err := Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var backend *SocketTransport
	done := make(chan struct{})
	go func() {
		backend, _ = Connect(sock)
		close(done)
	}()
	shell, _ := ln.Accept()
	<-done

	// Close shell side.
	shell.Close()

	// Backend should get an error on next recv.
	_, err = backend.RecvFromShell()
	if err == nil {
		t.Fatal("expected error after shell closed")
	}
	backend.Close()
}

func TestSocketTransport_Reconnect(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "reconnect.sock")

	ln, err := Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// First connection.
	var backend1 *SocketTransport
	done := make(chan struct{})
	go func() {
		backend1, _ = Connect(sock)
		close(done)
	}()
	shell1, _ := ln.Accept()
	<-done

	// Exchange a message.
	backend1.SendToShell(BackendMsg{Type: BackendReady, Version: ProtocolVersion}) //nolint:errcheck
	msg, _ := shell1.RecvFromBackend()
	if msg.Type != BackendReady {
		t.Fatal("first connection should work")
	}

	// Disconnect first backend.
	backend1.Close()
	shell1.Close()

	// Second connection — simulates hot-swap.
	var backend2 *SocketTransport
	done2 := make(chan struct{})
	go func() {
		backend2, _ = Connect(sock)
		close(done2)
	}()
	shell2, _ := ln.Accept()
	<-done2
	defer shell2.Close()
	defer backend2.Close()

	// Second connection should work.
	backend2.SendToShell(BackendMsg{Type: BackendReady, Version: ProtocolVersion}) //nolint:errcheck
	msg2, err := shell2.RecvFromBackend()
	if err != nil {
		t.Fatal(err)
	}
	if msg2.Type != BackendReady {
		t.Fatal("reconnected backend should work")
	}
}

func TestSocketTransport_ConcurrentSend(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "concurrent.sock")

	ln, err := Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var backend *SocketTransport
	done := make(chan struct{})
	go func() {
		backend, _ = Connect(sock)
		close(done)
	}()
	shell, _ := ln.Accept()
	defer shell.Close()
	<-done
	defer backend.Close()

	// 10 goroutines sending concurrently from backend.
	const n = 10
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(idx int) {
			defer wg.Done()
			backend.SendToShell(BackendMsg{Type: BackendText, Text: "msg"}) //nolint:errcheck
		}(i)
	}

	// Receive all.
	received := 0
	errCh := make(chan error, 1)
	go func() {
		for range n {
			_, err := shell.RecvFromBackend()
			if err != nil {
				errCh <- err
				return
			}
			received++
		}
		errCh <- nil
	}()

	wg.Wait()
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	if received != n {
		t.Fatalf("received %d, want %d", received, n)
	}
}

func TestSocketTransport_DoubleClose(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "dblclose.sock")

	ln, err := Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var backend *SocketTransport
	done := make(chan struct{})
	go func() {
		backend, _ = Connect(sock)
		close(done)
	}()
	shell, _ := ln.Accept()
	<-done

	shell.Close()
	shell.Close() // should not panic

	backend.Close()
	backend.Close() // should not panic
}

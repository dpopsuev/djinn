package ari

import (
	"testing"
	"time"

	ariTypes "github.com/dpopsuev/djinn/ari"
	"github.com/dpopsuev/djinn/orchestrator"
)

func TestMockARI_RoundTrip(t *testing.T) {
	server := NewMockARIServer()
	client := NewMockARIClient(server)

	// Register intent handler that emits events and a result
	server.OnIntent(func(intent ariTypes.Intent) {
		server.EmitProgress(orchestrator.Event{
			ExecID: intent.ID,
			Kind:   orchestrator.StageStarted,
			Stage:  "code",
		})
		server.EmitProgress(orchestrator.Event{
			ExecID: intent.ID,
			Kind:   orchestrator.StageCompleted,
			Stage:  "code",
		})
		server.EmitResult(ariTypes.Result{
			IntentID: intent.ID,
			Success:  true,
			Summary:  "done",
		})
	})

	// Client sends intent
	client.SendIntent(ariTypes.Intent{
		ID:     "int-1",
		Action: "fix-bug",
	})

	// Client waits for events
	event, ok := client.WaitForEvent(orchestrator.StageStarted, time.Second)
	if !ok {
		t.Fatal("timed out waiting for StageStarted")
	}
	if event.ExecID != "int-1" {
		t.Fatalf("event.ExecID = %q, want %q", event.ExecID, "int-1")
	}

	// Client waits for result
	result, ok := client.WaitForResult(time.Second)
	if !ok {
		t.Fatal("timed out waiting for result")
	}
	if !result.Success {
		t.Fatal("result.Success = false, want true")
	}
	if result.IntentID != "int-1" {
		t.Fatalf("result.IntentID = %q, want %q", result.IntentID, "int-1")
	}
}

func TestMockARI_PermissionRoundTrip(t *testing.T) {
	server := NewMockARIServer()
	client := NewMockARIClient(server)

	// Server emits permission request
	server.EmitPermission(ariTypes.PermissionPayload{
		ExecID:      "exec-1",
		Stage:       "deploy",
		Description: "deploy to prod",
	})

	// Client responds
	client.RespondPermission(ariTypes.PermissionResponse{
		ExecID:   "exec-1",
		Approved: true,
	})

	// Server reads response
	resp := <-server.PermissionResponses()
	if !resp.Approved {
		t.Fatal("resp.Approved = false, want true")
	}
	if resp.ExecID != "exec-1" {
		t.Fatalf("resp.ExecID = %q, want %q", resp.ExecID, "exec-1")
	}
}

func TestMockARI_WaitForEventTimeout(t *testing.T) {
	server := NewMockARIServer()
	client := NewMockARIClient(server)

	_, ok := client.WaitForEvent(orchestrator.ExecutionDone, 50*time.Millisecond)
	if ok {
		t.Fatal("expected timeout, got event")
	}
}

func TestMockARI_WaitForResultTimeout(t *testing.T) {
	server := NewMockARIServer()
	client := NewMockARIClient(server)

	_, ok := client.WaitForResult(50 * time.Millisecond)
	if ok {
		t.Fatal("expected timeout, got result")
	}
}

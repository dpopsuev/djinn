package agent

import (
	"encoding/json"
	"testing"
)

func TestTranslatePath_FilePathField(t *testing.T) {
	input := json.RawMessage(`{"file_path":"/home/user/project/main.go"}`)
	result := TranslatePath(input, "/home/user/project", "/workspace")

	var params map[string]string
	json.Unmarshal(result, &params)
	if params["file_path"] != "/workspace/main.go" {
		t.Fatalf("file_path = %q, want /workspace/main.go", params["file_path"])
	}
}

func TestTranslatePath_PathField(t *testing.T) {
	input := json.RawMessage(`{"path":"/home/user/project/src/lib.go"}`)
	result := TranslatePath(input, "/home/user/project", "/workspace")

	var params map[string]string
	json.Unmarshal(result, &params)
	if params["path"] != "/workspace/src/lib.go" {
		t.Fatalf("path = %q", params["path"])
	}
}

func TestTranslatePath_NoMatch(t *testing.T) {
	input := json.RawMessage(`{"file_path":"/etc/hosts"}`)
	result := TranslatePath(input, "/home/user/project", "/workspace")

	var params map[string]string
	json.Unmarshal(result, &params)
	if params["file_path"] != "/etc/hosts" {
		t.Fatal("should not translate paths outside workspace")
	}
}

func TestTranslatePath_EmptyInput(t *testing.T) {
	result := TranslatePath(nil, "/home", "/workspace")
	if result != nil {
		t.Fatal("nil input should return nil")
	}
}

func TestTranslatePath_EmptyWorkDir(t *testing.T) {
	input := json.RawMessage(`{"file_path":"/home/user/main.go"}`)
	result := TranslatePath(input, "", "/workspace")
	if string(result) != string(input) {
		t.Fatal("empty workdir should return input unchanged")
	}
}

package execution_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/kejne/gremlins/internal/execution"
	"github.com/kejne/gremlins/internal/results"
)

func TestWithWorkspaceRemovesTemporaryRootAfterRun(t *testing.T) {
	var path string
	_, err := execution.WithWorkspace(func(workspace string) (results.Results, error) {
		path = workspace
		if !strings.HasPrefix(path, os.TempDir()+string(os.PathSeparator)+"gremlins-") {
			t.Fatalf("workspace path %q is not under the Gremlins temporary root", path)
		}
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Fatalf("workspace root is not an existing directory: info=%v err=%v", info, err)
		}

		return results.Results{}, nil
	})
	if err != nil {
		t.Fatalf("run with workspace: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("workspace still exists after run: %v", err)
	}
}

func TestWithWorkspaceCleansAfterRunError(t *testing.T) {
	wantErr := errors.New("run failed")
	var path string
	_, err := execution.WithWorkspace(func(workspace string) (results.Results, error) {
		path = workspace

		return results.Results{}, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("run error = %v, want %v", err, wantErr)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("workspace still exists after failed run: %v", err)
	}
}

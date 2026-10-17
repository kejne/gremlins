package diff_test

import (
	"errors"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/kejne/gremlins/internal/diff"
)

func TestNewWithCmd(t *testing.T) {
	t.Run("empty reference returns nil without running command", func(t *testing.T) {
		result := &commandResult{}

		got, err := diff.NewWithCmd(commandFor(result), "")

		if err != nil || got != nil {
			t.Fatalf("NewWithCmd() = (%v, %v), want (nil, nil)", got, err)
		}
		if result.calls != 0 {
			t.Errorf("command called %d times, want 0", result.calls)
		}
	})

	t.Run("parses changed ranges across files and ignores deletions", func(t *testing.T) {
		result := &commandResult{output: []byte(`diff --git a/one.go b/one.go
--- a/one.go
+++ b/one.go
@@ -1,3 +1,4 @@
 first
+inserted
 second
 third
diff --git a/two.go b/two.go
--- a/two.go
+++ b/two.go
@@ -7,2 +7,1 @@
-old
 remaining
`)}

		got, err := diff.NewWithCmd(commandFor(result), "base-ref")
		if err != nil {
			t.Fatalf("NewWithCmd() error = %v", err)
		}

		for _, test := range []struct {
			name string
			pos  token.Position
			want bool
		}{
			{name: "inserted line selected", pos: token.Position{Filename: "one.go", Line: 2}, want: true},
			{name: "neighboring unchanged line excluded", pos: token.Position{Filename: "one.go", Line: 3}, want: false},
			{name: "deletion does not select remaining line", pos: token.Position{Filename: "two.go", Line: 7}, want: false},
			{name: "missing file excluded", pos: token.Position{Filename: "missing.go", Line: 2}, want: false},
		} {
			t.Run(test.name, func(t *testing.T) {
				if changed := got.IsChanged(test.pos); changed != test.want {
					t.Errorf("IsChanged(%+v) = %v, want %v", test.pos, changed, test.want)
				}
			})
		}

		if result.calls != 1 || result.name != "git" || !reflect.DeepEqual(result.args, []string{"diff", "--merge-base", "base-ref"}) {
			t.Errorf("command invocation = (%q, %v), calls %d; want git diff --merge-base base-ref once", result.name, result.args, result.calls)
		}
	})

	t.Run("command failure includes command output and error", func(t *testing.T) {
		commandErr := errors.New("git failed")
		result := &commandResult{output: []byte("diagnostic"), err: commandErr}

		_, err := diff.NewWithCmd(commandFor(result), "base-ref")
		if !errors.Is(err, commandErr) {
			t.Fatalf("NewWithCmd() error = %v, want wrapped command error", err)
		}
		if got := err.Error(); !strings.Contains(got, "diagnostic") {
			t.Errorf("error %q does not include command output", got)
		}
	})

	t.Run("malformed diff returns an error", func(t *testing.T) {
		result := &commandResult{output: []byte(`diff --git a/one.go b/one.go
--- a/one.go
+++ b/one.go
@@ -1,7 +1,7 @@
 context
+inserted
 context
`)}

		if _, err := diff.NewWithCmd(commandFor(result), "base-ref"); err == nil {
			t.Fatal("NewWithCmd() error = nil, want parse error")
		}
	})
}

type commandResult struct {
	output []byte
	err    error
	calls  int
	name   string
	args   []string
}

func (r *commandResult) CombinedOutput() ([]byte, error) {
	return r.output, r.err
}

func commandFor(result *commandResult) func(string, ...string) *commandResult {
	return func(name string, args ...string) *commandResult {
		result.calls++
		result.name = name
		result.args = append([]string(nil), args...)

		return result
	}
}

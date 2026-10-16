package diff

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/bluekeyes/go-gitdiff/gitdiff"

	"github.com/kejne/gremlins/internal/log"
)

// New creates a new Diff by parsing git diff output using the default command executor.
func New(diffRef string) (Diff, error) {
	return NewWithCmd(exec.Command, diffRef)
}

type execCmd interface {
	CombinedOutput() ([]byte, error)
}

// NewWithCmd creates a new Diff by parsing git diff output using a custom command executor.
// This is useful for testing.
func NewWithCmd[T execCmd](cmdContext func(name string, args ...string) T, diffRef string) (Diff, error) {
	if diffRef == "" {
		return nil, nil
	}

	log.Infoln("Gathering files diff...")

	cmd := cmdContext("git", "diff", "--merge-base", diffRef)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("an error occured while calling git diff: %w\n\n%s", err, out)
	}

	files, _, err := gitdiff.Parse(bytes.NewReader(out))
	if err != nil {
		return nil, fmt.Errorf("an error occured while parsing diff: %w", err)
	}

	return newDiff(files), nil
}

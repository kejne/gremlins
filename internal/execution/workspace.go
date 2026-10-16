package execution

import (
	"fmt"
	"os"

	"github.com/kejne/gremlins/internal/log"
	"github.com/kejne/gremlins/internal/results"
)

// WithWorkspace runs work inside a temporary root and removes the root when the run completes.
func WithWorkspace(run func(path string) (results.Results, error)) (results.Results, error) {
	path, err := os.MkdirTemp(os.TempDir(), "gremlins-")
	if err != nil {
		return results.Results{}, fmt.Errorf("impossible to create the workdir: %w", err)
	}
	defer cleanWorkspace(path)

	return run(path)
}

func cleanWorkspace(path string) {
	if err := os.RemoveAll(path); err != nil {
		log.Errorf("impossible to remove temporary folder: %s\n\t%s", err, path)
	}
}

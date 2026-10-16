package results

import (
	"errors"
	"go/token"
	"math"
	"testing"

	"github.com/kejne/gremlins/internal/mutator"
)

type mutableMutator struct {
	position          token.Position
	original, mutated []byte
}

func (m *mutableMutator) Type() mutator.Type       { return mutator.ConditionalsBoundary }
func (m *mutableMutator) SetType(mutator.Type)     {}
func (m *mutableMutator) Status() mutator.Status   { return mutator.Killed }
func (m *mutableMutator) SetStatus(mutator.Status) {}
func (m *mutableMutator) Position() token.Position { return m.position }
func (m *mutableMutator) Pos() token.Pos           { return token.NoPos }
func (m *mutableMutator) Pkg() string              { return "" }
func (m *mutableMutator) SetWorkdir(string)        {}
func (m *mutableMutator) Workdir() string          { return "" }
func (m *mutableMutator) Apply() error             { return nil }
func (m *mutableMutator) Rollback() error          { return nil }
func (m *mutableMutator) OrigSnippet() []byte      { return m.original }
func (m *mutableMutator) MutatedSnippet() []byte   { return m.mutated }

func TestSnapshotDetachesMutableState(t *testing.T) {
	m := &mutableMutator{position: token.Position{Filename: "a.go", Line: 2, Column: 3}, original: []byte("old"), mutated: []byte("new")}
	outcome := Snapshot(m)
	m.position.Filename = "changed.go"
	m.original[0] = 'X'
	m.mutated[0] = 'Y'
	if outcome.Position.Filename != "a.go" || string(outcome.OriginalSnippet) != "old" || string(outcome.MutatedSnippet) != "new" {
		t.Fatalf("outcome changed after mutation: %#v", outcome)
	}
}

func TestAssessStatisticsAndThresholds(t *testing.T) {
	outcomes := []Outcome{
		{Status: mutator.Killed}, {Status: mutator.Lived}, {Status: mutator.NotCovered},
		{Status: mutator.Skipped}, {Status: mutator.TimedOut}, {Status: mutator.NotViable},
	}
	stats, err := Assess(outcomes, Settings{ThresholdEfficacy: 50})
	if !errors.Is(err, ErrEfficacyBelowThreshold) {
		t.Fatalf("assessment error = %v", err)
	}
	if stats.Killed != 1 || stats.Lived != 1 || stats.NotCovered != 1 || stats.Skipped != 1 || stats.TimedOut != 1 || stats.NotViable != 1 || stats.TestEfficacy != 50 || math.Abs(stats.MutantCoverage-2.0/3*100) > 1e-9 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestAssessDryRunIgnoresThresholdAndUsesRunnableCoverage(t *testing.T) {
	stats, err := Assess([]Outcome{{Status: mutator.Runnable}, {Status: mutator.NotCovered}}, Settings{DryRun: true, ThresholdMutantCoverage: 100})
	if err != nil {
		t.Fatalf("dry-run assessment failed: %v", err)
	}
	if stats.Runnable != 1 || stats.MutantCoverage != 50 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

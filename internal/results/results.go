// Package results owns completed mutation outcomes and quality-gate assessment.
package results

import (
	"errors"
	"time"

	"github.com/kejne/gremlins/internal/mutator"
)

// Outcome is a detached, value-only snapshot of a completed mutation.
type Outcome struct {
	Position        Position
	Type            mutator.Type
	Status          mutator.Status
	OriginalSnippet []byte
	MutatedSnippet  []byte
}

// Position contains the source location fields needed by outcome consumers.
type Position struct {
	Filename string
	Offset   int
	Line     int
	Column   int
}

// Snapshot copies all mutable data exposed by a mutator into an ordinary outcome value.
func Snapshot(m mutator.Mutant) Outcome {
	p := m.Position()

	return Outcome{
		Position: Position{Filename: p.Filename, Offset: p.Offset, Line: p.Line, Column: p.Column},
		Type:     m.Type(), Status: m.Status(),
		OriginalSnippet: append([]byte(nil), m.OrigSnippet()...),
		MutatedSnippet:  append([]byte(nil), m.MutatedSnippet()...),
	}
}

// Results is a completed run independent of executable mutation objects.
type Results struct {
	Module   string
	Outcomes []Outcome
	Elapsed  time.Duration
}

// Settings controls assessment without depending on presentation.
type Settings struct {
	DryRun                  bool
	ThresholdEfficacy       float64
	ThresholdMutantCoverage float64
}

// Stats contains the values calculated from completed outcomes.
type Stats struct {
	Killed, Lived, TimedOut, NotCovered, Skipped, NotViable, Runnable int
	TestEfficacy, MutantCoverage                                      float64
	MutatorCounts                                                     map[mutator.Type]int
}

// ErrEfficacyBelowThreshold identifies an efficacy quality-gate failure.
var ErrEfficacyBelowThreshold = errors.New("below efficacy-threshold")

// ErrCoverageBelowThreshold identifies a mutation-coverage quality-gate failure.
var ErrCoverageBelowThreshold = errors.New("below mutant coverage-threshold")

// Assess calculates statistics and reports any configured quality-gate failure.
func Assess(outcomes []Outcome, settings Settings) (Stats, error) {
	stats := Stats{MutatorCounts: make(map[mutator.Type]int)}
	for _, outcome := range outcomes {
		stats.MutatorCounts[outcome.Type]++
		switch outcome.Status {
		case mutator.Killed:
			stats.Killed++
		case mutator.Lived:
			stats.Lived++
		case mutator.TimedOut:
			stats.TimedOut++
		case mutator.NotCovered:
			stats.NotCovered++
		case mutator.Skipped:
			stats.Skipped++
		case mutator.NotViable:
			stats.NotViable++
		case mutator.Runnable:
			stats.Runnable++
		}
	}
	if !settings.DryRun {
		if stats.Killed > 0 {
			stats.TestEfficacy = float64(stats.Killed) / float64(stats.Killed+stats.Lived) * 100
		}
		if stats.Killed+stats.Lived > 0 {
			stats.MutantCoverage = float64(stats.Killed+stats.Lived) / float64(stats.Killed+stats.Lived+stats.NotCovered) * 100
		}
	} else if stats.Runnable > 0 {
		stats.MutantCoverage = float64(stats.Runnable) / float64(stats.Runnable+stats.NotCovered) * 100
	}
	if settings.DryRun {
		return stats, nil
	}
	if settings.ThresholdEfficacy > 0 && stats.Killed+stats.Lived > 0 && stats.TestEfficacy <= settings.ThresholdEfficacy {
		return stats, ErrEfficacyBelowThreshold
	}
	if settings.ThresholdMutantCoverage > 0 && stats.Killed+stats.Lived+stats.NotCovered > 0 && stats.MutantCoverage <= settings.ThresholdMutantCoverage {
		return stats, ErrCoverageBelowThreshold
	}

	return stats, nil
}

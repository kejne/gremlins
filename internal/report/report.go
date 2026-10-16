/*
 * Copyright 2022 The Gremlins Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package report

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aymanbagabas/go-udiff"
	"github.com/fatih/color"
	"github.com/hako/durafmt"

	"github.com/kejne/gremlins/internal/log"
	"github.com/kejne/gremlins/internal/mutator"
	"github.com/kejne/gremlins/internal/report/internal"
	resultvalues "github.com/kejne/gremlins/internal/results"
)

var (
	fgRed      = color.New(color.FgRed).SprintFunc()
	fgGreen    = color.New(color.FgGreen).SprintFunc()
	fgHiGreen  = color.New(color.FgHiGreen).SprintFunc()
	fgHiBlack  = color.New(color.FgHiBlack).SprintFunc()
	fgHiYellow = color.New(color.FgYellow).SprintFunc()
)

// Results is the value-only result of a completed run.
type Results = resultvalues.Results

// Settings contains report output options resolved by the CLI.
type Settings struct {
	DryRun bool
	Output string
}

type reportStatus struct {
	files map[string][]internal.Mutation

	elapsed *durafmt.Durafmt
	module  string

	mutatorStatistics internal.MutatorType

	stats    resultvalues.Stats
	settings Settings
}

func newReport(results Results, stats resultvalues.Stats, settings Settings) (*reportStatus, bool) {
	if len(results.Outcomes) == 0 {
		return nil, false
	}
	rep := &reportStatus{
		module:   results.Module,
		elapsed:  durafmt.Parse(results.Elapsed).LimitFirstN(2),
		settings: settings,
	}
	rep.files = make(map[string][]internal.Mutation)
	for _, outcome := range results.Outcomes {
		rep.files[outcome.Position.Filename] = append(rep.files[outcome.Position.Filename], internal.Mutation{
			Line: outcome.Position.Line, Column: outcome.Position.Column,
			Type: outcome.Type.String(), Status: outcome.Status.String(),
		})
	}
	rep.stats = stats
	rep.mutatorStatistics = toMutatorStatistics(rep.stats.MutatorCounts)

	return rep, true
}

func toMutatorStatistics(counts map[mutator.Type]int) internal.MutatorType {
	return internal.MutatorType{
		ArithmeticBase:           counts[mutator.ArithmeticBase],
		ConditionalsNegation:     counts[mutator.ConditionalsNegation],
		ConditionalsBoundary:     counts[mutator.ConditionalsBoundary],
		IncrementDecrement:       counts[mutator.IncrementDecrement],
		InvertAssignments:        counts[mutator.InvertAssignments],
		InvertBitwise:            counts[mutator.InvertBitwise],
		InvertBitwiseAssignments: counts[mutator.InvertBitwiseAssignments],
		InvertLogical:            counts[mutator.InvertLogical],
		InvertLoopCtrl:           counts[mutator.InvertLoopCtrl],
		InvertNegatives:          counts[mutator.InvertNegatives],
		RemoveSelfAssignments:    counts[mutator.RemoveSelfAssignments],
	}
}

func (r *reportStatus) isDryRun() bool {
	return r.settings.DryRun
}

func (r *reportStatus) reportFindings() {
	if r.isDryRun() {
		r.dryRunReport()
	} else {
		r.fullRunReport()
	}
	r.fileReport()
}

func (r *reportStatus) fileReport() {
	if output := r.settings.Output; output != "" {
		files := make([]internal.OutputFile, 0, len(r.files))
		for fName, mutations := range r.files {
			of := internal.OutputFile{Filename: fName}
			of.Mutations = append(of.Mutations, mutations...)
			files = append(files, of)
		}

		result := internal.OutputResult{
			GoModule:          r.module,
			TestEfficacy:      r.stats.TestEfficacy,
			MutationsCoverage: r.stats.MutantCoverage,
			MutantsTotal:      r.stats.Lived + r.stats.Killed + r.stats.NotViable,
			MutantsKilled:     r.stats.Killed,
			MutantsLived:      r.stats.Lived,
			MutantsNotViable:  r.stats.NotViable,
			MutantsNotCovered: r.stats.NotCovered,
			ElapsedTime:       r.elapsed.Duration().Seconds(),
			MutatorStatistics: r.mutatorStatistics,
			Files:             files,
		}

		jsonResult, _ := json.Marshal(result)
		//nolint:gosec // output path is from configuration, not arbitrary user input
		f, err := os.Create(output)
		if err != nil {
			log.Errorf("impossible to write file: %s\n", err)
		}
		defer func(f *os.File) {
			_ = f.Close()
		}(f)
		if _, err := f.Write(jsonResult); err != nil {
			log.Errorf("impossible to write file: %s\n", err)
		}

	}
}

func (r *reportStatus) dryRunReport() {
	notCovered := fgHiYellow(r.stats.NotCovered)
	runnable := fgGreen(r.stats.Runnable)
	log.Infoln("")
	log.Infof("Dry run completed in %s\n", r.elapsed.String())
	log.Infof("Runnable: %s, Not covered: %s\n", runnable, notCovered)
	log.Infof("Mutator coverage: %.2f%%\n", r.stats.MutantCoverage)
}

func (r *reportStatus) fullRunReport() {
	killed := fgHiGreen(r.stats.Killed)
	lived := fgRed(r.stats.Lived)
	timedOut := fgGreen(r.stats.TimedOut)
	notViable := fgHiBlack(r.stats.NotViable)
	skipped := fgHiBlack(r.stats.Skipped)
	notCovered := fgHiYellow(r.stats.NotCovered)
	log.Infoln("")
	log.Infof("Mutation testing completed in %s\n", r.elapsed.String())
	log.Infof("Killed: %s, Lived: %s, Not covered: %s\n", killed, lived, notCovered)
	log.Infof("Timed out: %s, Not viable: %s, Skipped: %s\n", timedOut, notViable, skipped)
	log.Infof("Test efficacy: %.2f%%\n", r.stats.TestEfficacy)
	log.Infof("Mutator coverage: %.2f%%\n", r.stats.MutantCoverage)
}

// Do generates the report of the Results received.
// This function uses the log package in gremlins to write to the
// chosen io.Writer, so it is necessary to call log.Init before
// the report generation.
func Do(results Results, stats resultvalues.Stats, settings Settings) error {
	rep, ok := newReport(results, stats, settings)
	if !ok {
		log.Infoln("\nNo results to report.")

		return nil
	}
	rep.reportFindings()

	return nil
}

// MutantOutcome logs a detached completed mutation outcome.
func MutantOutcome(outcome resultvalues.Outcome) {
	status := outcome.Status.String()
	switch outcome.Status {
	case mutator.Killed, mutator.Runnable:
		status = fgHiGreen(outcome.Status)
	case mutator.Lived:
		status = fgRed(outcome.Status)
	case mutator.NotCovered:
		status = fgHiYellow(outcome.Status)
	case mutator.TimedOut:
		status = fgGreen(outcome.Status)
	case mutator.NotViable, mutator.Skipped:
		status = fgHiBlack(outcome.Status)
	}
	position := fmt.Sprintf("%s:%d:%d", outcome.Position.Filename, outcome.Position.Line, outcome.Position.Column)
	log.Infof("%s%s %s at %s\n", padding(outcome.Status), status, outcome.Type, position)
}

// MutantOutcomeDiff logs the unified diff stored in a completed outcome.
func MutantOutcomeDiff(outcome resultvalues.Outcome) {
	if len(outcome.OriginalSnippet) == 0 || len(outcome.MutatedSnippet) == 0 {
		return
	}
	diffLines, err := generateSnippetDiff(outcome.OriginalSnippet, outcome.MutatedSnippet)
	if err != nil {
		log.Errorf("failed to generate diff for mutant: %s", err.Error())

		return
	}
	for _, line := range diffLines {
		log.Infof("%s%s\n", padding(outcome.Status), line)
	}
}

// generateSnippetDiff returns the unified diff between original and mutated source snippets.
func generateSnippetDiff(original, mutated []byte) ([]string, error) {
	if len(original) == 0 || len(mutated) == 0 {
		return nil, nil
	}

	diffText, err := udiff.ToUnified("Original", "Mutated", string(original), udiff.Bytes(original, mutated), 3)
	if err != nil {
		return nil, fmt.Errorf("failed to generate diff: %w", err)
	}

	lines := strings.Split(diffText, "\n")
	if len(lines) <= 3 {
		return nil, nil
	}

	diffBuilder := make([]string, 0, len(lines)-3)
	for _, line := range lines {
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "---"), strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "@@"):
			continue
		case strings.Contains(line, "No newline at end of file"):
			continue
		case strings.HasPrefix(line, "-"):
			diffBuilder = append(diffBuilder, fgRed(line))
		case strings.HasPrefix(line, "+"):
			diffBuilder = append(diffBuilder, fgGreen(line))
		default:
			diffBuilder = append(diffBuilder, fgHiBlack(line))
		}
	}

	return diffBuilder, nil
}

func padding(s mutator.Status) string {
	var pad string
	padLen := 12 - len(s.String())
	for i := 0; i < padLen; i++ {
		pad += " "
	}

	return pad
}

/*
 * Copyright 2022 The Gremlins Authors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package engine_test

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/kejne/gremlins/internal/configuration"
	"github.com/kejne/gremlins/internal/coverage"
	"github.com/kejne/gremlins/internal/diff"
	"github.com/kejne/gremlins/internal/engine"
	"github.com/kejne/gremlins/internal/engine/workerpool"
	"github.com/kejne/gremlins/internal/gomodule"
	"github.com/kejne/gremlins/internal/log"
	"github.com/kejne/gremlins/internal/mutator"
	"github.com/kejne/gremlins/internal/report"
	resultvalues "github.com/kejne/gremlins/internal/results"
)

func TestWorkflowDiscoversExecutesAndAssessesMutations(t *testing.T) {
	viperSet(map[string]any{
		configuration.UnleashDryRunKey:             false,
		configuration.UnleashWorkersKey:            1,
		configuration.UnleashThresholdEfficacyKey:  float64(75),
		configuration.UnleashThresholdMCoverageKey: float64(0),
	})
	defer viperReset()
	log.Init(&bytes.Buffer{}, &bytes.Buffer{})
	defer log.Reset()

	const source = "package sample\nfunc Less(a, b int) bool {\n\treturn a < b\n}\n"
	const sourceFile = "sample.go"
	workflowFS := fstest.MapFS{
		sourceFile:       {Data: []byte(source)},
		"sample_test.go": {Data: []byte("package sample\n")},
		"notes.txt":      {Data: []byte("<")},
	}
	coverageProfile := coverage.Profile{
		sourceFile: {{StartLine: 3, EndLine: 3, StartCol: 11, EndCol: 11}},
	}

	workflow := engine.New(
		gomodule.GoModule{Name: "example.com/sample", Root: ".", CallingDir: "."},
		engine.CodeData{
			Cov:  coverageProfile,
			Diff: diff.Diff{diff.FileName(sourceFile): {{StartLine: 3, EndLine: 3}}},
		},
		controlledOutcomeDealer{},
		engine.Settings{EnabledMutators: map[mutator.Type]bool{
			mutator.ConditionalsBoundary: true,
			mutator.ConditionalsNegation: true,
		}, Workers: 1},
		engine.WithDirFs(workflowFS),
	)
	results, err := workflow.Run(context.Background())
	if err != nil {
		t.Fatalf("workflow run failed: %v", err)
	}

	if got, want := len(results.Outcomes), 2; got != want {
		t.Fatalf("discovered %d mutations, want %d", got, want)
	}
	statuses := map[mutator.Type]mutator.Status{}
	for _, mutation := range results.Outcomes {
		statuses[mutation.Type] = mutation.Status
		if mutation.Position.Filename != sourceFile {
			t.Errorf("mutation discovered outside source file: %s", mutation.Position.Filename)
		}
		if mutation.Status != mutator.Killed && mutation.Status != mutator.Lived {
			t.Errorf("mutation %s has outcome %s", mutation.Type, mutation.Status)
		}
	}
	if got := statuses[mutator.ConditionalsBoundary]; got != mutator.Killed {
		t.Errorf("conditional-boundary outcome = %s, want killed", got)
	}
	if got := statuses[mutator.ConditionalsNegation]; got != mutator.Lived {
		t.Errorf("conditional-negation outcome = %s, want lived", got)
	}

	err = report.Do(results, report.Settings{ThresholdEfficacy: 75})
	if !errors.Is(err, resultvalues.ErrEfficacyBelowThreshold) {
		t.Fatalf("assessment error = %v, want efficacy-threshold", err)
	}
}

type controlledOutcomeDealer struct{}

func (controlledOutcomeDealer) NewExecutor(mutation mutator.Mutator, out chan<- mutator.Mutator, wg *sync.WaitGroup) workerpool.Executor {
	return controlledOutcomeExecutor{mutation: mutation, out: out, wg: wg}
}

type controlledOutcomeExecutor struct {
	mutation mutator.Mutator
	out      chan<- mutator.Mutator
	wg       *sync.WaitGroup
}

func (executor controlledOutcomeExecutor) Start(_ *workerpool.Worker) {
	if executor.mutation.Status() == mutator.Runnable {
		if executor.mutation.Type() == mutator.ConditionalsBoundary {
			executor.mutation.SetStatus(mutator.Killed)
		} else {
			executor.mutation.SetStatus(mutator.Lived)
		}
	}
	executor.out <- executor.mutation
	executor.wg.Done()
}

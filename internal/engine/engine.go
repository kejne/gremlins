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

// Package engine orchestrates mutation testing by discovering, applying, and testing mutations.
package engine

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kejne/gremlins/internal/coverage"
	"github.com/kejne/gremlins/internal/diff"
	"github.com/kejne/gremlins/internal/engine/workerpool"
	"github.com/kejne/gremlins/internal/exclusion"
	"github.com/kejne/gremlins/internal/mutator"
	"github.com/kejne/gremlins/internal/report"
	"github.com/kejne/gremlins/internal/results"

	"github.com/kejne/gremlins/internal/gomodule"
)

// Engine is the "engine" that performs the mutation testing.
//
// It traverses the AST of the project, finds which TokenMutator can be applied and
// performs the actual mutation testing.
type Engine struct {
	fs           fs.FS
	jDealer      ExecutorDealer
	codeData     CodeData
	module       gomodule.GoModule
	logger       report.MutantLogger
	settings     Settings
	poolSettings workerpool.Settings
}

// Settings contains mutation-discovery options resolved by the CLI.
type Settings struct {
	EnabledMutators map[mutator.Type]bool
	Workers         int
	IntegrationMode bool
}

// CodeData is used to check if the mutant should be executed.
type CodeData struct {
	Cov       coverage.Profile
	Diff      diff.Diff
	Exclusion exclusion.Rules
}

// Option for the Engine initialization.
type Option func(m Engine) Engine

// New instantiates an Engine.
//
// It gets a fs.FS on which to perform the analysis, a CodeData to
// check if the mutants are executable and a sets of Option.
func New(mod gomodule.GoModule, codeData CodeData, jDealer ExecutorDealer, settings Settings, opts ...Option) Engine {
	dirFS := os.DirFS(filepath.Join(mod.Root, mod.CallingDir))
	mut := Engine{
		module:       mod,
		jDealer:      jDealer,
		codeData:     codeData,
		fs:           dirFS,
		logger:       report.MutantLogger{},
		settings:     settings,
		poolSettings: workerpool.Settings{Workers: settings.Workers, IntegrationMode: settings.IntegrationMode},
	}
	for _, opt := range opts {
		mut = opt(mut)
	}

	return mut
}

// WithLogger sets the configured mutation progress logger.
func WithLogger(logger report.MutantLogger) Option {
	return func(m Engine) Engine {
		m.logger = logger

		return m
	}
}

// WithDirFs overrides the fs.FS of the module (mainly used for testing purposes).
func WithDirFs(dirFS fs.FS) Option {
	return func(m Engine) Engine {
		m.fs = dirFS

		return m
	}
}

// Run discovers and executes mutations, returning discovery failures to the caller.
//
// It checks every non-test .go file in the provided fs.FS and scans it for token mutations.
func (mu *Engine) Run(ctx context.Context) (results.Results, error) {
	start := time.Now()
	mutants, err := mu.discoverMutants(ctx)
	if ctx.Err() != nil {
		return results.Results{}, ctx.Err()
	}
	if err != nil {
		return results.Results{}, fmt.Errorf("discover mutations: %w", err)
	}

	res := mu.executeTests(ctx, mutants)
	res.Elapsed = time.Since(start)
	res.Module = mu.module.Name

	return res, nil
}

func (mu *Engine) discoverMutants(ctx context.Context) ([]mutator.Mutator, error) {
	var mutants []mutator.Mutator
	err := fs.WalkDir(mu.fs, ".", func(path string, _ fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return fmt.Errorf("walk %q: %w", path, walkErr)
		}

		isGoCode := filepath.Ext(path) == ".go" && !strings.HasSuffix(path, "_test.go")
		if !isGoCode || mu.codeData.Exclusion.IsFileExcluded(path) {
			return nil
		}

		found, err := mu.runOnFile(path)
		if err != nil {
			return err
		}
		mutants = append(mutants, found...)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return mutants, nil
}

func (mu *Engine) runOnFile(fileName string) ([]mutator.Mutator, error) {
	src, err := fs.ReadFile(mu.fs, fileName)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", fileName, err)
	}
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, fileName, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", fileName, err)
	}

	var mutants []mutator.Mutator
	ast.Inspect(file, func(node ast.Node) bool {
		n, ok := NewTokenNode(node)
		if !ok {
			return true
		}
		mutants = append(mutants, mu.findMutations(fileName, set, file, n)...)

		return true
	})

	return mutants, nil
}

func (mu *Engine) findMutations(fileName string, set *token.FileSet, file *ast.File, node *NodeToken) []mutator.Mutator {
	mutantTypes, ok := TokenMutantType[node.Tok()]
	if !ok {
		return nil
	}

	pkg := mu.pkgName(fileName, file.Name.Name)
	var mutants []mutator.Mutator
	for _, mt := range mutantTypes {
		if !mu.settings.EnabledMutators[mt] {
			continue
		}
		mutantType := mt
		tm := NewTokenMutant(pkg, set, file, node)
		tm.SetType(mutantType)
		tm.SetStatus(mu.mutationStatus(set.Position(node.TokPos)))

		mutants = append(mutants, tm)
	}

	return mutants
}

func (mu *Engine) pkgName(fileName, fPkg string) string {
	var pkg string
	fn := fmt.Sprintf("%s/%s", mu.module.CallingDir, fileName)
	p := filepath.Dir(fn)
	for {
		if strings.HasSuffix(p, fPkg) {
			pkg = fmt.Sprintf("%s/%s", mu.module.Name, p)

			break
		}
		d := filepath.Dir(p)
		if d == p {
			pkg = mu.module.Name

			break
		}
		p = d
	}

	return normalisePkgPath(pkg)
}

func normalisePkgPath(pkg string) string {
	sep := fmt.Sprintf("%c", os.PathSeparator)

	return strings.ReplaceAll(pkg, sep, "/")
}

func (mu *Engine) mutationStatus(pos token.Position) mutator.Status {
	var status mutator.Status

	if mu.codeData.Cov.IsCovered(pos) {
		status = mutator.Runnable
	}

	if !mu.codeData.Diff.IsChanged(pos) {
		status = mutator.Skipped
	}

	return status
}

func (mu *Engine) executeTests(ctx context.Context, discovered []mutator.Mutator) results.Results {
	pool := workerpool.Initialize("mutator", mu.poolSettings)
	pool.Start()

	var mutants []mutator.Mutator
	outCh := make(chan mutator.Mutator)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, mut := range discovered {
			ok := checkDone(ctx)
			if !ok {
				pool.Stop()

				break
			}
			wg.Add(1)
			pool.AppendExecutor(mu.jDealer.NewExecutor(mut, outCh, wg))
		}
	}()

	go func() {
		wg.Wait()
		close(outCh)
	}()

	for m := range outCh {
		mu.logger.Mutant(m)
		mutants = append(mutants, m)
	}

	return completedResults(mutants)
}

func checkDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	default:
		return true
	}
}

func completedResults(m []mutator.Mutator) results.Results {
	outcomes := make([]results.Outcome, 0, len(m))
	for _, mutant := range m {
		outcomes = append(outcomes, results.Snapshot(mutant))
	}

	return results.Results{Outcomes: outcomes}
}

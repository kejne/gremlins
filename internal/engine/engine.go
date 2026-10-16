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
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/kejne/gremlins/internal/coverage"
	"github.com/kejne/gremlins/internal/diff"
	"github.com/kejne/gremlins/internal/exclusion"
	"github.com/kejne/gremlins/internal/execution"
	"github.com/kejne/gremlins/internal/mutation"
	"github.com/kejne/gremlins/internal/mutator"
	"github.com/kejne/gremlins/internal/results"

	"github.com/kejne/gremlins/internal/gomodule"
)

// Engine is the "engine" that performs the mutation testing.
//
// It traverses the AST of the project, finds which TokenMutant can be applied and
// performs the actual mutation testing.
type Engine struct {
	fs       fs.FS
	runner   execution.Runner
	codeData CodeData
	module   gomodule.GoModule
	progress func(results.Outcome)
	settings Settings
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
func New(mod gomodule.GoModule, codeData CodeData, runner execution.Runner, settings Settings, opts ...Option) Engine {
	dirFS := os.DirFS(filepath.Join(mod.Root, mod.CallingDir))
	mut := Engine{
		module:   mod,
		runner:   runner,
		codeData: codeData,
		fs:       dirFS,
		settings: settings,
	}
	for _, opt := range opts {
		mut = opt(mut)
	}

	return mut
}

// WithProgress sets a consumer for each completed mutation outcome.
func WithProgress(consume func(results.Outcome)) Option {
	return func(m Engine) Engine {
		m.progress = consume

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
	mutants, err := mutation.Discover(ctx, mu.fs, mu.module, mutation.DiscoveryData{
		Coverage: mu.codeData.Cov, Diff: mu.codeData.Diff, Exclusion: mu.codeData.Exclusion,
	}, mutation.DiscoverySettings{EnabledMutators: mu.settings.EnabledMutators})
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

func (mu *Engine) executeTests(ctx context.Context, discovered []mutator.Mutator) results.Results {
	completed := mu.runner.Run(ctx, discovered, execution.SchedulingSettings{
		Workers: mu.settings.Workers, IntegrationMode: mu.settings.IntegrationMode,
	}, mu.progress)

	return completed
}

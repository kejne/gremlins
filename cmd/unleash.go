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

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/kejne/gremlins/cmd/internal/flags"
	"github.com/kejne/gremlins/internal/configuration"
	"github.com/kejne/gremlins/internal/coverage"
	"github.com/kejne/gremlins/internal/diff"
	"github.com/kejne/gremlins/internal/engine"
	"github.com/kejne/gremlins/internal/engine/workdir"
	"github.com/kejne/gremlins/internal/exclusion"
	"github.com/kejne/gremlins/internal/gomodule"
	"github.com/kejne/gremlins/internal/log"
	"github.com/kejne/gremlins/internal/mutator"
	"github.com/kejne/gremlins/internal/report"
)

type unleashCmd struct {
	cmd *cobra.Command
}

const (
	commandName = "unleash"

	paramDiff               = "diff"
	paramBuildTags          = "tags"
	paramCoverPackages      = "coverpkg"
	paramDryRun             = "dry-run"
	paramOutputStatuses     = "output-statuses"
	paramOutputDiffStatuses = "output-diff-statuses"
	paramOutput             = "output"
	paramIntegrationMode    = "integration"
	paramExcludeFiles       = "exclude-files"
	paramTestCPU            = "test-cpu"
	paramWorkers            = "workers"
	paramTimeoutCoefficient = "timeout-coefficient"

	// Thresholds.
	paramThresholdEfficacy  = "threshold-efficacy"
	paramThresholdMCoverage = "threshold-mcover"
)

func newUnleashCmd(ctx context.Context) (*unleashCmd, error) {
	cmd := &cobra.Command{
		Use:     fmt.Sprintf("%s [path]", commandName),
		Aliases: []string{"run", "r"},
		Args:    cobra.MaximumNArgs(1),
		Short:   "Unleash the gremlins",
		Long:    longExplainer(),
		RunE:    runUnleash(ctx),
	}

	if err := setFlagsOnCmd(cmd); err != nil {
		return nil, err
	}

	return &unleashCmd{cmd: cmd}, nil
}

func longExplainer() string {
	return heredoc.Doc(`
		Unleashes the gremlins and performs mutation testing on a Go module. It works by
		first gathering the coverage of the test suite and then analysing the source
		code to look for supported mutants.

		Unleash only tests covered mutants, since it doesn't make sense to test mutants 
		that no test case is able to catch.

		In 'dry-run' mode, unleash only performs the analysis of the source code, but it
		doesn't actually perform the test.

		Thresholds are configurable quality gates that make gremlins exit with an error 
		if those values are not met. Efficacy is the percent of KILLED mutants over
		the total KILLED and LIVED mutants. Mutant coverage is the percent of total
		KILLED + LIVED mutants, over the total mutants.
	`)
}

func runUnleash(ctx context.Context) func(cmd *cobra.Command, args []string) error {
	return func(_ *cobra.Command, args []string) error {
		log.Infoln("Starting...")
		path, _ := os.Getwd()
		if len(args) > 0 {
			path = args[0]
		}
		mod, err := gomodule.Init(path)
		if err != nil {
			return fmt.Errorf("not in a Go module: %w", err)
		}

		workDir, err := os.MkdirTemp(os.TempDir(), "gremlins-")
		if err != nil {
			return fmt.Errorf("impossible to create the workdir: %w", err)
		}
		defer cleanUp(workDir)

		settings := resolveRunSettings()
		wg := &sync.WaitGroup{}
		wg.Add(1)
		cancelled := false
		var results report.Results
		go runWithCancel(ctx, wg, func(c context.Context) {
			results, err = run(c, mod, workDir, settings.coverage, settings.engine, settings.executor, settings.logger, settings.diffRef, settings.exclusions)
		}, func() {
			cancelled = true
		})
		wg.Wait()
		if cancelled || errors.Is(err, context.Canceled) {
			return nil
		}
		if err != nil {
			return err
		}

		return report.Do(results, settings.report)
	}
}

func runWithCancel(ctx context.Context, wg *sync.WaitGroup, runner func(c context.Context), onCancel func()) {
	c, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()
		log.Infof("\nShutting down gracefully...\n")
		cancel()
		onCancel()
	}()
	runner(c)
	wg.Done()
}

func cleanUp(wd string) {
	if err := os.RemoveAll(wd); err != nil {
		log.Errorf("impossible to remove temporary folder: %s\n\t%s", err, wd)
	}
}

type runSettings struct {
	coverage   coverage.Settings
	engine     engine.Settings
	executor   engine.ExecutorSettings
	report     report.Settings
	logger     report.LoggerSettings
	diffRef    string
	exclusions []string
}

func threshold(key string) float64 {
	value := configuration.Get[float64](key)
	if value == 0 {
		value = float64(configuration.Get[int](key))
	}

	return value
}

func resolveRunSettings() runSettings {
	integration := configuration.Get[bool](configuration.UnleashIntegrationMode)
	buildTags := configuration.Get[string](configuration.UnleashTagsKey)
	dryRun := configuration.Get[bool](configuration.UnleashDryRunKey)
	thresholdEfficacy := threshold(configuration.UnleashThresholdEfficacyKey)
	thresholdCoverage := threshold(configuration.UnleashThresholdMCoverageKey)
	enabled := make(map[mutator.Type]bool)
	for _, typ := range mutator.Types {
		key := configuration.MutantTypeEnabledKey(typ)
		if viper.IsSet(key) {
			enabled[typ] = configuration.Get[bool](key)
		} else {
			enabled[typ] = configuration.IsDefaultEnabled(typ)
		}
	}

	return runSettings{
		coverage:   coverage.Settings{BuildTags: buildTags, CoverPkg: configuration.Get[string](configuration.UnleashCoverPkgKey), IntegrationMode: integration},
		engine:     engine.Settings{EnabledMutators: enabled, Workers: configuration.Get[int](configuration.UnleashWorkersKey), IntegrationMode: integration},
		executor:   engine.ExecutorSettings{BuildTags: buildTags, DryRun: dryRun, IntegrationMode: integration, TestCPU: configuration.Get[int](configuration.UnleashTestCPUKey), TimeoutCoefficient: configuration.Get[int](configuration.UnleashTimeoutCoefficientKey)},
		report:     report.Settings{DryRun: dryRun, Output: configuration.Get[string](configuration.UnleashOutputKey), ThresholdEfficacy: thresholdEfficacy, ThresholdMutantCoverage: thresholdCoverage},
		logger:     report.LoggerSettings{OutputStatuses: configuration.Get[string](configuration.UnleashOutputStatusesKey), OutputDiffStatuses: configuration.Get[string](configuration.UnleashOutputDiffStatusesKey)},
		diffRef:    configuration.Get[string](configuration.UnleashDiffRef),
		exclusions: viper.GetStringSlice(configuration.UnleashExcludeFiles),
	}
}

func run(ctx context.Context, mod gomodule.GoModule, workDir string, coverageSettings coverage.Settings, engineSettings engine.Settings, executorSettings engine.ExecutorSettings, loggerSettings report.LoggerSettings, diffRef string, exclusionPatterns []string) (report.Results, error) {
	fDiff, err := diff.New(diffRef)
	if err != nil {
		return report.Results{}, err
	}

	c := coverage.New(workDir, mod, coverageSettings)

	exclude, err := exclusion.New(exclusionPatterns)
	if err != nil {
		return report.Results{}, err
	}

	cProfile, err := c.Run()
	if err != nil {
		return report.Results{}, fmt.Errorf("failed to gather coverage: %w", err)
	}

	wdDealer := workdir.NewCachedDealer(workDir, mod.Root)
	defer wdDealer.Clean()

	jDealer := engine.NewExecutorDealer(mod, wdDealer, cProfile.Elapsed, executorSettings)

	codeData := engine.CodeData{
		Cov:       cProfile.Profile,
		Diff:      fDiff,
		Exclusion: exclude,
	}

	mut := engine.New(mod, codeData, jDealer, engineSettings, engine.WithLogger(report.NewLogger(loggerSettings)))
	results, err := mut.Run(ctx)
	if err != nil {
		return report.Results{}, err
	}

	return results, nil
}

func setFlagsOnCmd(cmd *cobra.Command) error {
	cmd.Flags().SortFlags = false
	cmd.Flags().SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		from := []string{".", "_"}
		to := "-"
		for _, sep := range from {
			name = strings.ReplaceAll(name, sep, to)
		}

		return pflag.NormalizedName(name)
	})

	fls := []*flags.Flag{
		{Name: paramDryRun, CfgKey: configuration.UnleashDryRunKey, Shorthand: "d", DefaultV: false, Usage: "find mutations but do not executes tests"},
		{Name: paramOutputStatuses, CfgKey: configuration.UnleashOutputStatusesKey, Shorthand: "S", DefaultV: "", Usage: "print only statuses from this flag, allowed values - 'lctkvsr'"},
		{Name: paramOutputDiffStatuses, CfgKey: configuration.UnleashOutputDiffStatusesKey, DefaultV: "", Usage: "print diff for mutants with these statuses, allowed values - 'lk'"},
		{Name: paramBuildTags, CfgKey: configuration.UnleashTagsKey, Shorthand: "t", DefaultV: "", Usage: "a comma-separated list of build tags"},
		{Name: paramCoverPackages, CfgKey: configuration.UnleashCoverPkgKey, DefaultV: "", Usage: "a comma-separated list of package patterns"},
		{Name: paramDiff, CfgKey: configuration.UnleashDiffRef, Shorthand: "D", DefaultV: "", Usage: "diff branch or commit"},
		{Name: paramOutput, CfgKey: configuration.UnleashOutputKey, Shorthand: "o", DefaultV: "", Usage: "set the output file for machine readable results"},
		{Name: paramIntegrationMode, CfgKey: configuration.UnleashIntegrationMode, Shorthand: "i", DefaultV: false, Usage: "makes Gremlins run the complete test suite for each mutation"},
		{Name: paramExcludeFiles, CfgKey: configuration.UnleashExcludeFiles, Shorthand: "E", DefaultV: []string{}, Usage: "exclude files from Gremlins run by filepath regexp"},
		{Name: paramThresholdEfficacy, CfgKey: configuration.UnleashThresholdEfficacyKey, DefaultV: float64(0), Usage: "threshold for code-efficacy percent"},
		{Name: paramThresholdMCoverage, CfgKey: configuration.UnleashThresholdMCoverageKey, DefaultV: float64(0), Usage: "threshold for mutant-coverage percent"},
		{Name: paramWorkers, CfgKey: configuration.UnleashWorkersKey, DefaultV: 0, Usage: "the number of workers to use in mutation testing"},
		{Name: paramTestCPU, CfgKey: configuration.UnleashTestCPUKey, DefaultV: 0, Usage: "the number of CPUs to allow each test run to use"},
		{Name: paramTimeoutCoefficient, CfgKey: configuration.UnleashTimeoutCoefficientKey, DefaultV: 0, Usage: "the coefficient by which the timeout is increased"},
	}

	for _, f := range fls {
		err := flags.Set(cmd, f)
		if err != nil {
			return err
		}
	}

	return setMutantTypeFlags(cmd)
}

func setMutantTypeFlags(cmd *cobra.Command) error {
	for _, mt := range mutator.Types {
		name := mt.String()
		usage := fmt.Sprintf("enable %q mutants", name)
		param := strings.ReplaceAll(name, "_", "-")
		param = strings.ToLower(param)
		confKey := configuration.MutantTypeEnabledKey(mt)

		err := flags.Set(cmd, &flags.Flag{
			Name:     param,
			CfgKey:   confKey,
			DefaultV: configuration.IsDefaultEnabled(mt),
			Usage:    usage,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

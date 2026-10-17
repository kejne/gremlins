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
	"fmt"
	"strings"
	"testing"

	"github.com/kejne/gremlins/internal/configuration"
	"github.com/kejne/gremlins/internal/mutator"
)

func TestResolveRunSettingsDefaults(t *testing.T) {
	configuration.Reset()
	t.Cleanup(configuration.Reset)

	settings := resolveRunSettings()
	if settings.coverage.IntegrationMode || settings.executor.IntegrationMode || settings.scheduling.IntegrationMode {
		t.Fatal("integration mode should default to false in all receiving modules")
	}
	if settings.coverage.BuildTags != "" || settings.executor.BuildTags != "" || settings.coverage.CoverPkg != "" {
		t.Fatal("build tags and cover packages should default to empty")
	}
	if settings.scheduling.Workers != 0 || settings.executor.TestCPU != 0 || settings.executor.TimeoutCoefficient != 0 {
		t.Fatal("worker and test execution settings should default to zero")
	}
	if settings.diffRef != "" || len(settings.exclusions) != 0 {
		t.Fatal("diff ref and exclusions should default to empty")
	}
	if settings.report.Output != "" || settings.logger.OutputStatuses != "" || settings.logger.OutputDiffStatuses != "" {
		t.Fatal("output settings should default to empty")
	}
	if settings.thresholdEfficacy != 0 || settings.thresholdMutantCoverage != 0 {
		t.Fatal("assessment thresholds should default to zero")
	}
	for _, typ := range mutator.Types {
		if got, want := settings.engine.EnabledMutators[typ], configuration.IsDefaultEnabled(typ); got != want {
			t.Errorf("mutator %s default = %v, want %v", typ, got, want)
		}
	}
}

func TestResolveRunSettingsPropagatesConfiguration(t *testing.T) {
	configuration.Reset()
	t.Cleanup(configuration.Reset)
	configuration.Set(configuration.UnleashIntegrationMode, true)
	configuration.Set(configuration.UnleashTagsKey, "integration,slow")
	configuration.Set(configuration.UnleashWorkersKey, 7)
	configuration.Set(configuration.UnleashTestCPUKey, 3)
	configuration.Set(configuration.UnleashTimeoutCoefficientKey, 4)
	configuration.Set(configuration.UnleashCoverPkgKey, "./...,example.com/other")
	configuration.Set(configuration.UnleashExcludeFiles, []string{`vendor/.*`, `generated/.*`})
	configuration.Set(configuration.UnleashDiffRef, "main")
	configuration.Set(configuration.UnleashOutputKey, "results.json")
	configuration.Set(configuration.UnleashOutputStatusesKey, "lk")
	configuration.Set(configuration.UnleashOutputDiffStatusesKey, "l")
	configuration.Set(configuration.UnleashThresholdEfficacyKey, float64(81))
	configuration.Set(configuration.UnleashThresholdMCoverageKey, float64(64))
	configuration.Set(configuration.MutantTypeEnabledKey(mutator.ArithmeticBase), false)

	settings := resolveRunSettings()
	if !settings.coverage.IntegrationMode || !settings.executor.IntegrationMode || !settings.scheduling.IntegrationMode {
		t.Error("integration mode was not propagated to coverage, executor, and scheduling")
	}
	if settings.coverage.BuildTags != "integration,slow" || settings.executor.BuildTags != "integration,slow" {
		t.Error("build tags were not propagated to coverage and executor")
	}
	if settings.scheduling.Workers != 7 || settings.executor.TestCPU != 3 || settings.executor.TimeoutCoefficient != 4 {
		t.Errorf("execution settings not propagated: scheduling=%+v executor=%+v", settings.scheduling, settings.executor)
	}
	if settings.coverage.CoverPkg != "./...,example.com/other" {
		t.Errorf("cover packages = %q", settings.coverage.CoverPkg)
	}
	if settings.diffRef != "main" || strings.Join(settings.exclusions, ",") != `vendor/.*,generated/.*` {
		t.Errorf("selection settings not propagated: diff=%q exclusions=%v", settings.diffRef, settings.exclusions)
	}
	if settings.report.Output != "results.json" || settings.logger.OutputStatuses != "lk" || settings.logger.OutputDiffStatuses != "l" {
		t.Errorf("reporting settings not propagated: report=%+v logger=%+v", settings.report, settings.logger)
	}
	if settings.thresholdEfficacy != 81 || settings.thresholdMutantCoverage != 64 {
		t.Errorf("assessment thresholds = efficacy %v, mutant coverage %v", settings.thresholdEfficacy, settings.thresholdMutantCoverage)
	}
	if settings.engine.EnabledMutators[mutator.ArithmeticBase] {
		t.Error("explicit mutator override was not honored")
	}
	if !settings.engine.EnabledMutators[mutator.ConditionalsBoundary] {
		t.Error("unspecified mutator should retain its configured default")
	}
}

func TestUnleash(t *testing.T) {
	c, err := newUnleashCmd(context.Background())
	if err != nil {
		t.Fatal("newUnleashCmd should no fail")
	}
	cmd := c.cmd

	if cmd.Name() != "unleash" {
		t.Errorf("expected 'unleash', got %q", cmd.Name())
	}

	flags := cmd.Flags()

	testCases := []struct {
		name      string
		shorthand string
		flagType  string
		defValue  string
	}{
		{
			name:     "arithmetic-base",
			flagType: "bool",
			defValue: "true",
		},
		{
			name:     "conditionals-boundary",
			flagType: "bool",
			defValue: "true",
		},
		{
			name:     "conditionals_negation",
			flagType: "bool",
			defValue: "true",
		},
		{
			name:     "coverpkg",
			flagType: "string",
			defValue: "",
		},
		{
			name:      "diff",
			shorthand: "D",
			flagType:  "string",
			defValue:  "",
		},
		{
			name:      "dry-run",
			shorthand: "d",
			flagType:  "bool",
			defValue:  "false",
		},
		{
			name:     "increment-decrement",
			flagType: "bool",
			defValue: "true",
		},
		{
			name:      "integration",
			shorthand: "i",
			flagType:  "bool",
			defValue:  "false",
		},
		{
			name:     "invert-assignments",
			flagType: "bool",
			defValue: "false",
		},
		{
			name:     "invert-bitwise",
			flagType: "bool",
			defValue: "false",
		},
		{
			name:     "invert-bwassign",
			flagType: "bool",
			defValue: "false",
		},

		{
			name:     "invert-logical",
			flagType: "bool",
			defValue: "false",
		},
		{
			name:     "invert-loopctrl",
			flagType: "bool",
			defValue: "false",
		},
		{
			name:     "invert-negatives",
			flagType: "bool",
			defValue: "true",
		},
		{
			name:      "output",
			shorthand: "o",
			flagType:  "string",
			defValue:  "",
		},
		{
			name:     "remove-self-assignments",
			flagType: "bool",
			defValue: "false",
		},
		{
			name:      "tags",
			shorthand: "t",
			flagType:  "string",
			defValue:  "",
		},
		{
			name:     "test-cpu",
			flagType: "int",
			defValue: "0",
		},
		{
			name:     "threshold-efficacy",
			flagType: "float64",
			defValue: "0",
		},
		{
			name:     "threshold-mcover",
			flagType: "float64",
			defValue: "0",
		},
		{
			name:     "timeout-coefficient",
			flagType: "int",
			defValue: "0",
		},
		{
			name:     "workers",
			flagType: "int",
			defValue: "0",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			f := flags.Lookup(tc.name)
			if f == nil {
				t.Fatalf("expected flag %q to be registered", tc.name)
			}
			if tc.shorthand != "" && f.Shorthand != tc.shorthand {
				t.Errorf("expected %q to have a shorthand %q, got %q", tc.name, tc.shorthand, f.Shorthand)
			}
			if f.Value.Type() != tc.flagType {
				t.Errorf("expected %q to be type %q, got %q", tc.name, f.Value.Type(), f.Value.Type())
			}
			if f.DefValue != tc.defValue {
				t.Errorf("expected %q to have default value %q, got %q", tc.name, tc.defValue, f.DefValue)
			}
		})
	}

	// test for MutantTypes flags
	for _, mt := range mutator.Types {
		s := strings.ToLower(mt.String())
		mtf := flags.Lookup(s)
		if mtf == nil {
			t.Errorf("expected to have flag for mutant type: %s", mt)

			continue
		}

		if mtf.Value.Type() != "bool" {
			t.Errorf("expected %q to be a %q, got %q", s, "bool", mtf.Value.Type())
		}
		wantDef := fmt.Sprintf("%v", configuration.IsDefaultEnabled(mt))
		if mtf.DefValue != wantDef {
			t.Errorf("expected %q have default %q, got %q", s, wantDef, mtf.DefValue)
		}
	}
}

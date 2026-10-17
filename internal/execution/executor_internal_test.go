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

package execution_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/kejne/gremlins/internal/execution"
	"github.com/kejne/gremlins/internal/execution/workdir"
	"github.com/kejne/gremlins/internal/gomodule"
	"github.com/kejne/gremlins/internal/mutation"
	"github.com/kejne/gremlins/internal/mutator"
)

func TestExecutorSettingsReachTestCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		settings execution.ExecutorSettings
		elapsed  time.Duration
		want     []string
	}{
		{name: "empty build tags and zero CPU", elapsed: time.Second, settings: execution.ExecutorSettings{}, want: []string{"test", "-timeout", "7s", "-failfast", "example.com/pkg"}},
		{name: "tags, CPU and package selection", elapsed: 3 * time.Second, settings: execution.ExecutorSettings{BuildTags: "one,two", TestCPU: 4}, want: []string{"test", "-tags", "one,two", "-timeout", "17s", "-failfast", "-cpu", "4", "example.com/pkg"}},
		{name: "integration selection", elapsed: time.Second, settings: execution.ExecutorSettings{IntegrationMode: true}, want: []string{"test", "-timeout", "7s", "-failfast", "./..."}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com\n\ngo 1.22\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			source := "package sample\nfunc f() int { return 1 + 2 }\n"
			if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, "sample.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			var binary *ast.BinaryExpr
			ast.Inspect(file, func(node ast.Node) bool {
				if expr, ok := node.(*ast.BinaryExpr); ok {
					binary = expr
				}

				return true
			})
			node, ok := mutation.NewTokenNode(binary)
			if !ok {
				t.Fatal("expected arithmetic mutation node")
			}
			mutant := mutation.NewTokenMutant("example.com/pkg", set, file, node)
			mutant.SetType(mutator.ArithmeticBase)
			mutant.SetStatus(mutator.Runnable)

			var got []string
			dealer := workdir.NewCachedDealer(t.TempDir(), root)
			defer dealer.Clean()
			executor := execution.NewExecutorDealer(gomodule.GoModule{Name: "example.com", Root: root, CallingDir: "."}, dealer, tc.elapsed, tc.settings,
				execution.WithExecContext(func(_ context.Context, name string, args ...string) *exec.Cmd {
					got = append([]string{name}, args...)

					return exec.Command("true")
				}))
			results := execution.NewRunner(executor, execution.SchedulingSettings{Workers: 1, IntegrationMode: tc.settings.IntegrationMode}).Run(context.Background(), []mutator.Mutant{mutant}, nil)
			if len(results.Outcomes) != 1 {
				t.Fatalf("expected one mutation outcome, got %d", len(results.Outcomes))
			}
			if !reflect.DeepEqual(got, append([]string{"go"}, tc.want...)) {
				t.Fatalf("command mismatch: got %q, want %q", got, append([]string{"go"}, tc.want...))
			}
		})
	}
}

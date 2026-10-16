package mutation_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/kejne/gremlins/internal/coverage"
	"github.com/kejne/gremlins/internal/diff"
	"github.com/kejne/gremlins/internal/exclusion"
	"github.com/kejne/gremlins/internal/gomodule"
	"github.com/kejne/gremlins/internal/mutation"
	"github.com/kejne/gremlins/internal/mutator"
)

func TestDiscoverUsesEnabledRulesAndSourceSelection(t *testing.T) {
	const file = "sample/source.go"
	source := fstest.MapFS{file: &fstest.MapFile{Data: []byte("package sample\nfunc add(a, b int) int { return a + b }\n")}}
	data := mutation.DiscoveryData{
		Coverage: coverage.Profile{file: {{StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 100}}},
		Diff:     diff.Diff{diff.FileName(file): {{StartLine: 2, EndLine: 2}}},
	}
	found, err := mutation.Discover(context.Background(), source, gomodule.GoModule{Name: "example.com/project", CallingDir: "."}, data,
		mutation.DiscoverySettings{EnabledMutators: map[mutator.Type]bool{mutator.ArithmeticBase: true}})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("Discover() found %d mutations, want 1", len(found))
	}
	if found[0].Type() != mutator.ArithmeticBase || found[0].Status() != mutator.Runnable {
		t.Fatalf("Discover() mutation = (%v, %v), want (%v, %v)", found[0].Type(), found[0].Status(), mutator.ArithmeticBase, mutator.Runnable)
	}
	if found[0].Pkg() != "example.com/project/sample" {
		t.Errorf("Discover() package = %q, want %q", found[0].Pkg(), "example.com/project/sample")
	}
}

func TestDiscoverExcludesTestsAndExcludedFiles(t *testing.T) {
	source := fstest.MapFS{
		"source.go":      &fstest.MapFile{Data: []byte("package sample\nfunc f(a,b int) int { return a+b }\n")},
		"source_test.go": &fstest.MapFile{Data: []byte("package sample\nfunc f(a,b int) int { return a+b }\n")},
		"skip.go":        &fstest.MapFile{Data: []byte("package sample\nfunc f(a,b int) int { return a+b }\n")},
	}
	rules, err := exclusion.New([]string{"skip.go"})
	if err != nil {
		t.Fatal(err)
	}
	found, err := mutation.Discover(context.Background(), source, gomodule.GoModule{Name: "example.com/project"}, mutation.DiscoveryData{Exclusion: rules}, mutation.DiscoverySettings{EnabledMutators: map[mutator.Type]bool{mutator.ArithmeticBase: true}})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("Discover() found %d mutations, want 1", len(found))
	}
}

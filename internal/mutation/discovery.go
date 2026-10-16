// Package mutation discovers Go AST mutation opportunities and applies them to source files.
package mutation

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

	"github.com/kejne/gremlins/internal/mutator"

	"github.com/kejne/gremlins/internal/coverage"
	"github.com/kejne/gremlins/internal/diff"
	"github.com/kejne/gremlins/internal/exclusion"
	"github.com/kejne/gremlins/internal/gomodule"
)

// DiscoverySettings controls which mutation rules are used during discovery.
type DiscoverySettings struct {
	EnabledMutators map[mutator.Type]bool
}

// DiscoveryData supplies concrete source-selection data to mutation discovery.
type DiscoveryData struct {
	Coverage  coverage.Profile
	Diff      diff.Diff
	Exclusion exclusion.Rules
}

// Discover finds enabled Go AST mutations that match coverage and changed-line data.
func Discover(ctx context.Context, source fs.FS, module gomodule.GoModule, data DiscoveryData, settings DiscoverySettings) ([]mutator.Mutator, error) {
	d := discovery{source: source, module: module, data: data, settings: settings}
	var mutations []mutator.Mutator
	err := fs.WalkDir(source, ".", func(path string, _ fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return fmt.Errorf("walk %q: %w", path, walkErr)
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") || data.Exclusion.IsFileExcluded(path) {
			return nil
		}
		found, err := d.runOnFile(path)
		if err != nil {
			return err
		}
		mutations = append(mutations, found...)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return mutations, nil
}

type discovery struct {
	source   fs.FS
	module   gomodule.GoModule
	data     DiscoveryData
	settings DiscoverySettings
}

func (d *discovery) runOnFile(fileName string) ([]mutator.Mutator, error) {
	src, err := fs.ReadFile(d.source, fileName)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", fileName, err)
	}
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, fileName, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %q: %w", fileName, err)
	}
	var mutations []mutator.Mutator
	ast.Inspect(file, func(node ast.Node) bool {
		n, ok := NewTokenNode(node)
		if ok {
			mutations = append(mutations, d.findMutations(fileName, set, file, n)...)
		}

		return true
	})

	return mutations, nil
}

func (d *discovery) findMutations(fileName string, set *token.FileSet, file *ast.File, node *NodeToken) []mutator.Mutator {
	types, ok := TokenMutantType[node.Tok()]
	if !ok {
		return nil
	}
	pkg := d.pkgName(fileName, file.Name.Name)
	var mutations []mutator.Mutator
	for _, mt := range types {
		if !d.settings.EnabledMutators[mt] {
			continue
		}
		m := NewTokenMutant(pkg, set, file, node)
		m.SetType(mt)
		m.SetStatus(d.mutationStatus(set.Position(node.TokPos)))
		mutations = append(mutations, m)
	}

	return mutations
}

func (d *discovery) pkgName(fileName, filePackage string) string {
	path := filepath.Dir(fmt.Sprintf("%s/%s", d.module.CallingDir, fileName))
	for {
		if strings.HasSuffix(path, filePackage) {
			return normalisePkgPath(fmt.Sprintf("%s/%s", d.module.Name, path))
		}
		next := filepath.Dir(path)
		if next == path {
			return d.module.Name
		}
		path = next
	}
}

func normalisePkgPath(pkg string) string {
	return strings.ReplaceAll(pkg, fmt.Sprintf("%c", os.PathSeparator), "/")
}

func (d *discovery) mutationStatus(pos token.Position) mutator.Status {
	status := mutator.NotCovered
	if d.data.Coverage.IsCovered(pos) {
		status = mutator.Runnable
	}
	if !d.data.Diff.IsChanged(pos) {
		status = mutator.Skipped
	}

	return status
}

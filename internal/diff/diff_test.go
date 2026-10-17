package diff_test

import (
	"go/token"
	"testing"

	"github.com/kejne/gremlins/internal/diff"
)

func TestDiffIsChanged(t *testing.T) {
	tests := []struct {
		name string
		diff diff.Diff
		pos  token.Position
		want bool
	}{
		{
			name: "nil diff includes every position",
			pos:  token.Position{Filename: "one.go", Line: 1},
			want: true,
		},
		{
			name: "empty diff includes every position",
			diff: diff.Diff{},
			pos:  token.Position{Filename: "one.go", Line: 1},
			want: true,
		},
		{
			name: "included range boundary",
			diff: diff.Diff{"one.go": {{StartLine: 21, EndLine: 23}}},
			pos:  token.Position{Filename: "one.go", Line: 23},
			want: true,
		},
		{
			name: "excluded range boundary",
			diff: diff.Diff{"one.go": {{StartLine: 21, EndLine: 23}}},
			pos:  token.Position{Filename: "one.go", Line: 24},
			want: false,
		},
		{
			name: "different file is excluded",
			diff: diff.Diff{"one.go": {{StartLine: 21, EndLine: 23}}},
			pos:  token.Position{Filename: "two.go", Line: 21},
			want: false,
		},
		{
			name: "files have independent changed ranges",
			diff: diff.Diff{
				"one.go": {{StartLine: 2, EndLine: 3}},
				"two.go": {{StartLine: 8, EndLine: 8}},
			},
			pos:  token.Position{Filename: "two.go", Line: 8},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diff.IsChanged(tt.pos); got != tt.want {
				t.Errorf("IsChanged(%+v) = %v, want %v", tt.pos, got, tt.want)
			}
		})
	}
}

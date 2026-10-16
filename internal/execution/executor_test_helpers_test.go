package execution

import (
	"errors"
	"go/token"
	"sync"
	"testing"

	"github.com/kejne/gremlins/internal/configuration"
	"github.com/kejne/gremlins/internal/mutator"
)

var viperMutex sync.RWMutex

func init() {
	viperMutex.Lock()
	viperReset()
}

func viperSet(values map[string]any) {
	viperMutex.Lock()
	for key, value := range values {
		configuration.Set(key, value)
	}
}

func viperReset() {
	configuration.Reset()
	for _, mt := range mutator.Types {
		configuration.Set(configuration.MutantTypeEnabledKey(mt), true)
	}
	viperMutex.Unlock()
}

type dealerStub struct {
	t     *testing.T
	fnGet func(string) (string, error)
}

func newWdDealerStub(t *testing.T) *dealerStub {
	t.Helper()

	return &dealerStub{t: t, fnGet: func(string) (string, error) { return t.TempDir(), nil }}
}
func (d dealerStub) Get(id string) (string, error) {
	if d.fnGet != nil {
		return d.fnGet(id)
	}

	return d.t.TempDir(), nil
}
func (dealerStub) Clean()          {}
func (dealerStub) WorkDir() string { return "/tmp" }

type mutantStub struct {
	worDir         string
	pkg            string
	position       token.Position
	status         mutator.Status
	mutType        mutator.Type
	applyCalled    bool
	rollbackCalled bool
	hasApplyError  bool
}

func (m *mutantStub) Type() mutator.Type         { return m.mutType }
func (m *mutantStub) SetType(mt mutator.Type)    { m.mutType = mt }
func (m *mutantStub) Status() mutator.Status     { return m.status }
func (m *mutantStub) SetStatus(s mutator.Status) { m.status = s }
func (m *mutantStub) Position() token.Position   { return m.position }
func (*mutantStub) Pos() token.Pos               { panic("not used in test") }
func (m *mutantStub) Pkg() string                { return m.pkg }
func (m *mutantStub) SetWorkdir(w string)        { m.worDir = w }
func (m *mutantStub) Workdir() string            { return m.worDir }
func (m *mutantStub) Apply() error {
	m.applyCalled = true
	if m.hasApplyError {
		return errors.New("test error")
	}

	return nil
}
func (m *mutantStub) Rollback() error {
	m.rollbackCalled = true

	return nil
}
func (*mutantStub) OrigSnippet() []byte    { return nil }
func (*mutantStub) MutatedSnippet() []byte { return nil }

func executorSettingsForTest() ExecutorSettings {
	return ExecutorSettings{
		BuildTags:          configuration.Get[string](configuration.UnleashTagsKey),
		DryRun:             configuration.Get[bool](configuration.UnleashDryRunKey),
		IntegrationMode:    configuration.Get[bool](configuration.UnleashIntegrationMode),
		TestCPU:            configuration.Get[int](configuration.UnleashTestCPUKey),
		TimeoutCoefficient: configuration.Get[int](configuration.UnleashTimeoutCoefficientKey),
	}
}

package gate

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestClosedError(t *testing.T) {
	var (
		assert       = assert.New(t)
		gate         = New(Config{Name: "test"})
		err    error = &ClosedError{
			Gate: gate,
		}
	)

	assert.NotEmpty(err.Error())
	assert.Contains(err.Error(), "test")
}

type GateTestSuite struct {
	suite.Suite
	gateName       string
	onOpenCalled   bool
	onClosedCalled bool
}

var _ suite.SetupAllSuite = (*GateTestSuite)(nil)
var _ suite.SetupTestSuite = (*GateTestSuite)(nil)

func (suite *GateTestSuite) SetupSuite() {
	suite.gateName = "test"
}

func (suite *GateTestSuite) SetupTest() {
	suite.resetCallbacks()
}

func (suite *GateTestSuite) onOpen(s Status) {
	suite.onOpenCalled = true
	suite.Equal(suite.gateName, s.Name())
	suite.True(s.IsOpen())
}

func (suite *GateTestSuite) onClosed(s Status) {
	suite.onClosedCalled = true
	suite.Equal(suite.gateName, s.Name())
	suite.False(s.IsOpen())
}

func (suite *GateTestSuite) resetCallbacks() {
	suite.onOpenCalled = false
	suite.onClosedCalled = false
}

func (suite *GateTestSuite) TestDefaults() {
	gate := New(Config{})
	suite.Require().NotNil(gate)

	suite.Empty(gate.Name())
	suite.Contains(fmt.Sprintf("%s", gate), gateOpenText)

	suite.True(gate.IsOpen())
	suite.False(gate.Open(), "Open should be idempotent")
	suite.True(gate.IsOpen(), "Open should be idempotent")

	suite.True(gate.Close())
	suite.Contains(fmt.Sprintf("%s", gate), gateClosedText)
	suite.False(gate.IsOpen())
	suite.False(gate.Close(), "Close should be idempotent")
	suite.False(gate.IsOpen(), "Close should be idempotent")

	suite.True(gate.Open())
	suite.Contains(fmt.Sprintf("%s", gate), gateOpenText)
	suite.True(gate.IsOpen())
	suite.False(gate.Open(), "Open should be idempotent")
	suite.True(gate.IsOpen())

	suite.Empty(gate.Name(), "Name should be immutable")
}

func (suite *GateTestSuite) TestInitiallyOpen() {
	gate := New(Config{
		Name: suite.gateName,
		Hooks: Hooks{
			{
				OnOpen:   suite.onOpen,
				OnClosed: suite.onClosed,
			},
		},
	})

	suite.Require().NotNil(gate)

	suite.T().Log("initial state")
	suite.Equal(suite.gateName, gate.Name())
	suite.True(gate.IsOpen())
	suite.Contains(fmt.Sprintf("%s", gate), gateOpenText)
	suite.True(suite.onOpenCalled)
	suite.False(suite.onClosedCalled)
	suite.resetCallbacks()

	suite.T().Log("opening an initially open gate should be idempotent")
	suite.False(gate.Open())
	suite.True(gate.IsOpen())
	suite.False(suite.onOpenCalled)
	suite.False(suite.onClosedCalled)
	suite.resetCallbacks()

	suite.T().Log("closing an open gate")
	suite.True(gate.Close())
	suite.Contains(fmt.Sprintf("%s", gate), gateClosedText)
	suite.False(gate.IsOpen())
	suite.False(suite.onOpenCalled)
	suite.True(suite.onClosedCalled)
	suite.resetCallbacks()
	suite.False(gate.Close())
	suite.False(gate.IsOpen())
	suite.False(suite.onOpenCalled)
	suite.False(suite.onClosedCalled)
	suite.resetCallbacks()

	suite.Equal(suite.gateName, gate.Name(), "Name should be immutable")
}

func (suite *GateTestSuite) TestInitiallyClosed() {
	gate := New(Config{
		Name:            suite.gateName,
		InitiallyClosed: true,
		Hooks: Hooks{
			{
				OnOpen:   suite.onOpen,
				OnClosed: suite.onClosed,
			},
		},
	})

	suite.Require().NotNil(gate)

	suite.T().Log("initial state")
	suite.Equal(suite.gateName, gate.Name())
	suite.False(gate.IsOpen())
	suite.Contains(fmt.Sprintf("%s", gate), gateClosedText)
	suite.False(suite.onOpenCalled)
	suite.True(suite.onClosedCalled)
	suite.resetCallbacks()

	suite.T().Log("closing an initially closed gate should be idempotent")
	suite.False(gate.Close())
	suite.False(gate.IsOpen())
	suite.False(suite.onOpenCalled)
	suite.False(suite.onClosedCalled)
	suite.resetCallbacks()

	suite.T().Log("opening a closed gate")
	suite.True(gate.Open())
	suite.Contains(fmt.Sprintf("%s", gate), gateOpenText)
	suite.True(gate.IsOpen())
	suite.True(suite.onOpenCalled)
	suite.False(suite.onClosedCalled)
	suite.resetCallbacks()
	suite.False(gate.Open())
	suite.True(gate.IsOpen())
	suite.False(suite.onOpenCalled)
	suite.False(suite.onClosedCalled)
	suite.resetCallbacks()

	suite.Equal(suite.gateName, gate.Name(), "Name should be immutable")
}

func (suite *GateTestSuite) TestInitiallyEmptyHook() {
	g := New(Config{
		Hooks: Hooks{
			{}, // no callbacks set
		},
	})

	suite.T().Log("configured hooks that are empty should have no effect")
	suite.True(g.Close())
	suite.True(g.Open())
}

func (suite *GateTestSuite) TestRegister() {
	suite.Run("OnOpen", func() {
		suite.resetCallbacks()
		g := New(Config{
			Name: suite.gateName,
		})

		g.Register(Hook{
			OnOpen:   suite.onOpen,
			OnClosed: suite.onClosed,
		})

		suite.True(suite.onOpenCalled, "Register should invoke OnOpen when the gate is open")
		suite.False(suite.onClosedCalled, "Register should not invoke OnClosed when the gate is open")

		suite.T().Log("callbacks should work normally after Register")
		suite.resetCallbacks()
		suite.True(g.Close())
		suite.False(suite.onOpenCalled)
		suite.True(suite.onClosedCalled)
	})

	suite.Run("OnClosed", func() {
		suite.resetCallbacks()
		g := New(Config{
			Name: suite.gateName,
		})

		suite.Require().True(g.Close())
		g.Register(Hook{
			OnOpen:   suite.onOpen,
			OnClosed: suite.onClosed,
		})

		suite.False(suite.onOpenCalled, "Register should not invoke OnOpen when the gate is closed")
		suite.True(suite.onClosedCalled, "Register should invoke OnClosed when the gate is closed")

		suite.T().Log("callbacks should work normally after Register")
		suite.resetCallbacks()
		suite.True(g.Open())
		suite.True(suite.onOpenCalled)
		suite.False(suite.onClosedCalled)
	})

	suite.Run("EmptyHook", func() {
		g := New(Config{})
		g.Register(Hook{})

		suite.T().Log("Register should do nothing when passed an empty Hook")
		suite.True(g.Close())
		suite.True(g.Open())
	})
}

func TestGate(t *testing.T) {
	suite.Run(t, new(GateTestSuite))
}

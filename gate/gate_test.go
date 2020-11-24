package httpaux

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type GateTestSuite struct {
	suite.Suite
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

func TestGate(t *testing.T) {
	suite.Run(t, new(GateTestSuite))
}

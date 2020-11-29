package gate

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ControlHandlerTestSuite struct {
	suite.Suite
}

func (suite *ControlHandlerTestSuite) noChange(*http.Request) StateChange   { return StateNoChange }
func (suite *ControlHandlerTestSuite) stateOpen(*http.Request) StateChange  { return StateOpen }
func (suite *ControlHandlerTestSuite) stateClose(*http.Request) StateChange { return StateClose }

func (suite *ControlHandlerTestSuite) Test() {
	cases := []struct {
		stateChange func(*http.Request) StateChange
		gate        Interface
		expectOpen  bool
	}{
		{
			stateChange: suite.noChange,
			gate:        New(Config{}),
			expectOpen:  true,
		},
		{
			stateChange: suite.stateOpen,
			gate:        New(Config{}),
			expectOpen:  true,
		},
		{
			stateChange: suite.stateOpen,
			gate: New(Config{
				InitiallyClosed: true,
			}),
			expectOpen: true,
		},
		{
			stateChange: suite.noChange,
			gate: New(Config{
				InitiallyClosed: true,
			}),
			expectOpen: false,
		},
		{
			stateChange: suite.stateClose,
			gate:        New(Config{}),
			expectOpen:  false,
		},
		{
			stateChange: suite.stateClose,
			gate: New(Config{
				InitiallyClosed: true,
			}),
			expectOpen: false,
		},
	}

	for i, testCase := range cases {
		suite.Run(strconv.Itoa(i), func() {
			response := httptest.NewRecorder()
			ControlHandler{
				StateChange: testCase.stateChange,
				Gate:        testCase.gate,
			}.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))

			suite.Equal(200, response.Code)
			suite.Equal(testCase.expectOpen, testCase.gate.IsOpen())
		})
	}
}

func TestControlHandler(t *testing.T) {
	suite.Run(t, new(ControlHandlerTestSuite))
}

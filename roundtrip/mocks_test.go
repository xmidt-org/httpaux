package roundtrip

import (
	"net/http"

	"github.com/stretchr/testify/mock"
)

type mockRoundTripper struct {
	mock.Mock
}

func (m *mockRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	arguments := m.Called(request)
	return arguments.Get(0).(*http.Response), arguments.Error(1)
}

type mockRoundTripperCloseIdler struct {
	mock.Mock
}

func (m *mockRoundTripperCloseIdler) RoundTrip(request *http.Request) (*http.Response, error) {
	arguments := m.Called(request)
	return arguments.Get(0).(*http.Response), arguments.Error(1)
}

func (m *mockRoundTripperCloseIdler) CloseIdleConnections() {
	m.Called()
}

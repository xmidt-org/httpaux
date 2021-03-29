package httpaux

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
)

type HeaderTestSuite struct {
	suite.Suite
}

func (suite *HeaderTestSuite) assertHeader(expected http.Header, actual Header) bool {
	actualHeader := make(http.Header)
	actual.SetTo(actualHeader)

	if expected == nil {
		return suite.Empty(actualHeader)
	}

	return suite.Equal(expected, actualHeader)
}

func (suite *HeaderTestSuite) requireHeader(expected http.Header, actual Header) {
	actualHeader := make(http.Header)
	actual.SetTo(actualHeader)

	if expected == nil {
		suite.Require().Empty(actualHeader)
	} else {
		suite.Require().Equal(expected, actualHeader)
	}
}

func (suite *HeaderTestSuite) TestEmptyHeader() {
	h := EmptyHeader()
	suite.Zero(h.Len())

	n := h.AppendHeaders("Header1", "value1")
	suite.NotSame(n, h)
	suite.assertHeader(
		http.Header{
			"Header1": {"value1"},
		},
		n,
	)

	// immutability
	suite.Zero(h.Len())
	suite.Zero(EmptyHeader().Len())
}

func (suite *HeaderTestSuite) TestAppend() {
	testCases := []struct {
		initial  http.Header
		toAppend http.Header
		expected http.Header
	}{
		{}, // everything empty
		{
			initial: http.Header{},
			toAppend: http.Header{
				"Header1": {"value1"},
			},
			expected: http.Header{
				"Header1": {"value1"},
			},
		},
		{
			initial: http.Header{
				"Initial1":    {"value1"},
				"Appended":    {"value1"},
				"EmptyValues": {}, // this shouldn't appear
			},
			toAppend: http.Header{
				"Appended":        {"value2"},
				"MoreEmptyValues": {}, // this shouldn't appear
				"Header1":         {"value1"},
				"Header2":         {"value1", "value2", "value3"},
			},
			expected: http.Header{
				"Initial1": {"value1"},
				"Appended": {"value1", "value2"},
				"Header1":  {"value1"},
				"Header2":  {"value1", "value2", "value3"},
			},
		},
	}

	for i, testCase := range testCases {
		suite.Run(strconv.Itoa(i), func() {
			h := NewHeader(testCase.initial)

			n := h.Append(testCase.toAppend)
			suite.assertHeader(testCase.expected, n)
		})
	}
}

func (suite *HeaderTestSuite) TestNewHeader() {
	testCases := []http.Header{
		{},
		{
			"Header1": {"value1"},
		},
		{
			"Header1": {"value1"},
			"Header2": {"value1", "value2", "value3"},
			"Header3": {"value1", "value2"},
		},
	}

	for i, testCase := range testCases {
		suite.Run(strconv.Itoa(i), func() {
			h := NewHeader(testCase)
			suite.assertHeader(testCase, h)
		})
	}
}

func (suite *HeaderTestSuite) TestAppendMap() {
	testCases := []struct {
		initial  map[string]string
		toAppend map[string]string
		expected http.Header
	}{
		{}, // everything empty
		{
			initial: map[string]string{},
			toAppend: map[string]string{
				"Header1": "value1",
			},
			expected: http.Header{
				"Header1": {"value1"},
			},
		},
		{
			initial: map[string]string{
				"Initial1": "value1",
				"Appended": "value1",
			},
			toAppend: map[string]string{
				"Appended": "value2",
				"Header1":  "value1",
			},
			expected: http.Header{
				"Initial1": {"value1"},
				"Appended": {"value1", "value2"},
				"Header1":  {"value1"},
			},
		},
	}

	for i, testCase := range testCases {
		suite.Run(strconv.Itoa(i), func() {
			h := NewHeaderFromMap(testCase.initial)

			n := h.AppendMap(testCase.toAppend)
			suite.assertHeader(testCase.expected, n)
		})
	}
}

func (suite *HeaderTestSuite) TestNewHeaderFromMap() {
	testCases := []struct {
		headers  map[string]string
		expected http.Header
	}{
		{}, // everything empty
		{
			headers: map[string]string{"Header1": "value1"},
			expected: http.Header{
				"Header1": {"value1"},
			},
		},
		{
			headers: map[string]string{
				"Header1": "value1",
				"Header2": "value2",
				"Header3": "value3",
			},
			expected: http.Header{
				"Header1": {"value1"},
				"Header2": {"value2"},
				"Header3": {"value3"},
			},
		},
	}

	for i, testCase := range testCases {
		suite.Run(strconv.Itoa(i), func() {
			h := NewHeaderFromMap(testCase.headers)
			suite.assertHeader(testCase.expected, h)
		})
	}
}

func TestHeader(t *testing.T) {
	suite.Run(t, new(HeaderTestSuite))
}

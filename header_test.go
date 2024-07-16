// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

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
				"initial1":    {"value1"}, // this should be canonicalized
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
				"header1": "value1", // this should be canonicalized
			},
			expected: http.Header{
				"Header1": {"value1"},
			},
		},
		{
			initial: map[string]string{
				"initial1": "value1", // this should be canonicalized
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
			headers: map[string]string{
				"Header1":                      "value1",
				"this-should-be-canonicalized": "value1",
			},
			expected: http.Header{
				"Header1":                      {"value1"},
				"This-Should-Be-Canonicalized": {"value1"},
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

func (suite *HeaderTestSuite) TestAppendHeaders() {
	testCases := []struct {
		initial  []string
		toAppend []string
		expected http.Header
	}{
		{}, // everything empty
		{
			initial:  []string{"header1", "value1", "HeaDer2", "value1"},
			toAppend: []string{"HEADER2", "value2", "blank"},
			expected: http.Header{
				"Header1": {"value1"},
				"Header2": {"value1", "value2"},
				"Blank":   {""},
			},
		},
		{
			initial:  []string{"Header1", "value1", "blank"},
			toAppend: []string{"BLANK", "value1"},
			expected: http.Header{
				"Header1": {"value1"},
				"Blank":   {"", "value1"},
			},
		},
	}

	for i, testCase := range testCases {
		suite.Run(strconv.Itoa(i), func() {
			h := NewHeaders(testCase.initial...)
			h = h.AppendHeaders(testCase.toAppend...)
			suite.assertHeader(testCase.expected, h)
		})
	}
}

func (suite *HeaderTestSuite) TestNewHeaders() {
	testCases := []struct {
		values   []string
		expected http.Header
	}{
		{}, // everything empty
		{
			values: []string{"Header1", "value1", "this-should-be-canonicalized", "value1"},
			expected: http.Header{
				"Header1":                      {"value1"},
				"This-Should-Be-Canonicalized": {"value1"},
			},
		},
		{
			values: []string{"Multivalue", "value1", "Header1", "value1", "Multivalue", "value2", "Blank"},
			expected: http.Header{
				"Header1":    {"value1"},
				"Multivalue": {"value1", "value2"},
				"Blank":      {""},
			},
		},
	}

	for i, testCase := range testCases {
		suite.Run(strconv.Itoa(i), func() {
			h := NewHeaders(testCase.values...)
			suite.assertHeader(testCase.expected, h)
		})
	}
}

func (suite *HeaderTestSuite) TestExtend() {
	testCases := []struct {
		initial  Header
		toExtend Header
		expected http.Header
	}{
		{}, // everything empty
		{
			initial:  Header{},
			toExtend: NewHeaders("header1", "value1"), // this should be canonicalized
			expected: http.Header{
				"Header1": {"value1"},
			},
		},
		{
			initial:  NewHeaders("Initial1", "value1", "Multivalue", "value1", "blank"),
			toExtend: NewHeaders("Multivalue", "value2", "Header1", "value1", "blank", "value1"),
			expected: http.Header{
				"Initial1":   {"value1"},
				"Header1":    {"value1"},
				"Multivalue": {"value1", "value2"},
				"Blank":      {"", "value1"},
			},
		},
	}

	for i, testCase := range testCases {
		suite.Run(strconv.Itoa(i), func() {
			h := testCase.initial.Extend(testCase.toExtend)
			suite.assertHeader(testCase.expected, h)
		})
	}
}

func (suite *HeaderTestSuite) TestSetTo() {
	testCases := []struct {
		header   Header
		expected http.Header
	}{
		{
			header:   Header{},
			expected: http.Header{},
		},
		{
			header: NewHeaders("Header1", "value1", "HeadEr2", "value2"),
			expected: http.Header{
				"Header1": {"value1"},
				"Header2": {"value2"},
			},
		},
	}

	for i, testCase := range testCases {
		suite.Run(strconv.Itoa(i), func() {
			actual := make(http.Header)
			testCase.header.SetTo(actual)
			suite.Equal(testCase.expected, actual)
		})
	}
}

func (suite *HeaderTestSuite) TestAddTo() {
	testCases := []struct {
		header   Header
		addTo    http.Header
		expected http.Header
	}{
		{
			header:   Header{},
			addTo:    http.Header{},
			expected: http.Header{},
		},
		{
			header: Header{},
			addTo: http.Header{
				"Header1": {"value1"},
			},
			expected: http.Header{
				"Header1": {"value1"},
			},
		},
		{
			header: NewHeaders("Header1", "value1", "header2", "value1"), // this should be canonicalized
			addTo:  http.Header{},
			expected: http.Header{
				"Header1": {"value1"},
				"Header2": {"value1"},
			},
		},
		{
			header: NewHeaders("Multivalue", "value2", "Initial1", "value1", "blank"),
			addTo: http.Header{
				"Multivalue": {"value1"},
			},
			expected: http.Header{
				"Multivalue": {"value1", "value2"},
				"Blank":      {""},
				"Initial1":   {"value1"},
			},
		},
	}

	for i, testCase := range testCases {
		suite.Run(strconv.Itoa(i), func() {
			testCase.header.AddTo(testCase.addTo)
			suite.Equal(testCase.expected, testCase.addTo)
		})
	}
}

func TestHeader(t *testing.T) {
	suite.Run(t, new(HeaderTestSuite))
}

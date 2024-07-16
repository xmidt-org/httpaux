// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package retry

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/httpaux/httpmock"
)

type StateTestSuite struct {
	suite.Suite
}

func (suite *StateTestSuite) TestPrepareNext() {
	s := &State{
		retries: 2,
	}

	suite.Run("Initial", func() {
		suite.Zero(s.Attempt())
		suite.Equal(2, s.Retries())
		actual, actualErr := s.Previous() //nolint:bodyclose
		suite.Nil(actual)
		suite.NoError(actualErr)
	})

	suite.Run("ErrorOnly", func() {
		expectedErr := errors.New("first")
		s.prepareNext(nil, expectedErr)
		suite.Equal(1, s.Attempt())
		suite.Equal(2, s.Retries())
		actual, actualErr := s.Previous() //nolint:bodyclose
		suite.Nil(actual)
		suite.Equal(expectedErr, actualErr)
	})

	suite.Run("NoBodyWithError", func() {
		expected := new(http.Response)
		expectedErr := errors.New("second")
		s.prepareNext(expected, expectedErr)
		suite.Equal(2, s.Attempt())
		suite.Equal(2, s.Retries())
		actual, actualErr := s.Previous() //nolint:bodyclose
		suite.True(expected == actual)
		suite.Equal(expectedErr, actualErr)
	})

	suite.Run("BodyWithoutError", func() {
		expected := &http.Response{
			Body: httpmock.EmptyBody(),
		}

		s.prepareNext(expected, nil)
		suite.Equal(3, s.Attempt()) // no attempt is made to keep attempt <= retries
		suite.Equal(2, s.Retries())
		actual, actualErr := s.Previous() //nolint:bodyclose
		suite.True(expected == actual)
		suite.NoError(actualErr)
		suite.Nil(actual.Body)
	})
}

func (suite *StateTestSuite) TestGetState() {
	suite.Run("Missing", func() {
		suite.Nil(
			GetState(context.Background()),
		)
	})

	suite.Run("Typical", func() {
		expected := new(State)
		ctx := withState(context.Background(), expected)
		suite.Require().NotNil(ctx)

		actual := GetState(ctx)
		suite.True(expected == actual)
	})
}

func TestState(t *testing.T) {
	suite.Run(t, new(StateTestSuite))
}

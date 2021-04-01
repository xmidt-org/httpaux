package erraux

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type FieldsTestSuite struct {
	suite.Suite
}

func (suite *FieldsTestSuite) TestNewFields() {
	suite.Run("ValidStatusCode", func() {
		f := NewFields(571, "here is a cause")
		suite.Equal(571, f.Code())
		suite.Equal("here is a cause", f.Cause())
	})

	suite.Run("InvalidStatusCode", func() {
		f := NewFields(15, "here is a cause")
		suite.Equal(http.StatusInternalServerError, f.Code())
		suite.Equal("here is a cause", f.Cause())
	})
}

func (suite *FieldsTestSuite) TestCode() {
	f := Fields{}
	suite.Zero(f.Code())

	f.SetCode(123)
	suite.Equal(123, f.Code())
}

func (suite *FieldsTestSuite) TestCause() {
	f := Fields{}
	suite.Empty(f.Cause())
	suite.False(f.HasCause())

	f.SetCause("here is a cause")
	suite.Equal("here is a cause", f.Cause())
	suite.True(f.HasCause())
}

func (suite *FieldsTestSuite) TestClone() {
	f := Fields{}

	clone := f.Clone()
	suite.Equal(f, clone)
	suite.NotSame(f, clone)

	f["foo"] = "bar"
	f["moo"] = 123

	clone = f.Clone()
	suite.Equal(f, clone)
	suite.NotSame(f, clone)
}

func (suite *FieldsTestSuite) TestMerge() {
	f := Fields{}

	toMerge := Fields{"foo": "bar"}
	f.Merge(toMerge)
	suite.Equal(
		Fields{"foo": "bar"},
		f,
	)

	suite.Equal(
		Fields{"foo": "bar"},
		toMerge,
		"Merge should not modify its argument",
	)

	toMerge = Fields{"moo": 123}
	f.Merge(toMerge)
	suite.Equal(
		Fields{
			"foo": "bar",
			"moo": 123,
		},
		f,
	)

	suite.Equal(
		Fields{"moo": 123},
		toMerge,
		"Merge should not modify its argument",
	)
}

func (suite *FieldsTestSuite) TestAdd() {
	f := Fields{}
	f.Add()
	suite.Empty(f)

	f.Add("foo", "bar", "moo", []int{1, 2})
	suite.Equal(
		Fields{
			"foo": "bar",
			"moo": []int{1, 2},
		},
		f,
	)

	f.Add("more", 67.8, "blank")
	suite.Equal(
		Fields{
			"foo":   "bar",
			"moo":   []int{1, 2},
			"more":  67.8,
			"blank": nil,
		},
		f,
	)
}

func (suite *FieldsTestSuite) TestAppend() {
	f := Fields{}

	nav := f.Append(nil)
	suite.Empty(nav)

	nav = f.Append([]interface{}{"foo", "bar"})
	suite.Equal(
		[]interface{}{"foo", "bar"},
		nav,
	)

	f["key"] = "value"
	f["age"] = 123
	nav = f.Append(nil)

	// have to reconstite into a map instead of comparing slices
	// because the iteration order for a golang map is not consistent
	appended := Fields{}
	appended.Add(nav...)
	suite.Equal(f, appended)

	nav = f.Append([]interface{}{"existing", true})
	appended = Fields{}
	appended.Add(nav...)
	clone := f.Clone()
	clone["existing"] = true
	suite.Equal(clone, appended)
}

func TestFields(t *testing.T) {
	suite.Run(t, new(FieldsTestSuite))
}

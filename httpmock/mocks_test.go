package httpmock

import "github.com/stretchr/testify/mock"

// testingT is a standin for a real *testing.T.  Used to
// test the test code.
//
// Rather than a mock, which would be problematic given
// all the different possibilities, this type just keeps
// track of how many of each test call was made.
type testingT struct {
	// T is the delegate that receive all method calls
	mock.TestingT

	Errors   int
	Failures int
}

var _ mock.TestingT = (*testingT)(nil)

func wrapTestingT(next mock.TestingT) *testingT {
	return &testingT{
		TestingT: next,
	}
}

func (t *testingT) Errorf(format string, args ...interface{}) {
	t.Errors++
	t.Logf("TEST ERRORF: "+format, args...)
	t.Logf("END TEST ERRORF")
}

func (t *testingT) FailNow() {
	t.Failures++
	t.Logf("TEST FAILNOW")
}

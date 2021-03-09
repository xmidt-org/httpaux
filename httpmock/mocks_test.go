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
	T mock.TestingT

	Logs     int
	Errors   int
	Failures int
}

var _ mock.TestingT = (*testingT)(nil)

func wrapTestingT(next mock.TestingT) *testingT {
	return &testingT{
		T: next,
	}
}

func (t *testingT) Logf(format string, args ...interface{}) {
	t.Logs++
	t.T.Logf("TEST LOGF: "+format, args...)
	t.T.Logf("END TEST LOGF")
}

func (t *testingT) Errorf(format string, args ...interface{}) {
	t.Errors++
	t.T.Logf("TEST ERRORF: "+format, args...)
	t.T.Logf("END TEST ERRORF")
}

func (t *testingT) FailNow() {
	t.Failures++
	t.T.Logf("TEST FAILNOW")
}

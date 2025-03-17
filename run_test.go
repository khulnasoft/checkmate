// These tests verify the test running logic.

package check_test

import (
	"errors"
	"sync"

	. "github.com/khulnasoft/checkmate"
)

var runnerS = Suite(&RunS{})

type RunS struct{}

func (s *RunS) TestCountSuite(c *C) {
	suitesRun += 1
}

// -----------------------------------------------------------------------
// Check result aggregation.

func (s *RunS) TestAdd(c *C) {
	result := &Result{
		Succeeded:        1,
		Skipped:          2,
		Failed:           3,
		Panicked:         4,
		FixturePanicked:  5,
		Missed:           6,
		ExpectedFailures: 7,
	}
	result.Add(&Result{
		Succeeded:        10,
		Skipped:          20,
		Failed:           30,
		Panicked:         40,
		FixturePanicked:  50,
		Missed:           60,
		ExpectedFailures: 70,
	})
	c.Check(result.Succeeded, Equals, 11)
	c.Check(result.Skipped, Equals, 22)
	c.Check(result.Failed, Equals, 33)
	c.Check(result.Panicked, Equals, 44)
	c.Check(result.FixturePanicked, Equals, 55)
	c.Check(result.Missed, Equals, 66)
	c.Check(result.ExpectedFailures, Equals, 77)
	c.Check(result.RunError, IsNil)
}

// -----------------------------------------------------------------------
// Check the Passed() method.

func (s *RunS) TestPassed(c *C) {
	c.Assert((&Result{}).Passed(), Equals, true)
	c.Assert((&Result{Succeeded: 1}).Passed(), Equals, true)
	c.Assert((&Result{Skipped: 1}).Passed(), Equals, true)
	c.Assert((&Result{Failed: 1}).Passed(), Equals, false)
	c.Assert((&Result{Panicked: 1}).Passed(), Equals, false)
	c.Assert((&Result{FixturePanicked: 1}).Passed(), Equals, false)
	c.Assert((&Result{Missed: 1}).Passed(), Equals, false)
	c.Assert((&Result{RunError: errors.New("!")}).Passed(), Equals, false)
}

// -----------------------------------------------------------------------
// Check that result printing is working correctly.

func (s *RunS) TestPrintSuccess(c *C) {
	result := &Result{Succeeded: 5}
	c.Check(result.String(), Equals, "OK: 5 passed")
}

func (s *RunS) TestPrintFailure(c *C) {
	result := &Result{Failed: 5}
	c.Check(result.String(), Equals, "OOPS: 0 passed, 5 FAILED")
}

func (s *RunS) TestPrintSkipped(c *C) {
	result := &Result{Skipped: 5}
	c.Check(result.String(), Equals, "OK: 0 passed, 5 skipped")
}

func (s *RunS) TestPrintExpectedFailures(c *C) {
	result := &Result{ExpectedFailures: 5}
	c.Check(result.String(), Equals, "OK: 0 passed, 5 expected failures")
}

func (s *RunS) TestPrintPanicked(c *C) {
	result := &Result{Panicked: 5}
	c.Check(result.String(), Equals, "OOPS: 0 passed, 5 PANICKED")
}

func (s *RunS) TestPrintFixturePanicked(c *C) {
	result := &Result{FixturePanicked: 5}
	c.Check(result.String(), Equals, "OOPS: 0 passed, 5 FIXTURE-PANICKED")
}

func (s *RunS) TestPrintMissed(c *C) {
	result := &Result{Missed: 5}
	c.Check(result.String(), Equals, "OOPS: 0 passed, 5 MISSED")
}

func (s *RunS) TestPrintAll(c *C) {
	result := &Result{Succeeded: 1, Skipped: 2, ExpectedFailures: 3,
		Panicked: 4, FixturePanicked: 5, Missed: 6}
	c.Check(result.String(), Equals,
		"OOPS: 1 passed, 2 skipped, 3 expected failures, 4 PANICKED, "+
			"5 FIXTURE-PANICKED, 6 MISSED")
}

func (s *RunS) TestPrintRunError(c *C) {
	result := &Result{Succeeded: 1, Failed: 1,
		RunError: errors.New("Kaboom!")}
	c.Check(result.String(), Equals, "ERROR: Kaboom!")
}

// -----------------------------------------------------------------------
// Verify that List works correctly.

func (s *RunS) TestListFiltered(c *C) {
	names := List(&FixtureHelper{}, &RunConf{Filter: "1"})
	c.Assert(names, DeepEquals, []string{
		"FixtureHelper.Test1",
	})
}

func (s *RunS) TestList(c *C) {
	names := List(&FixtureHelper{}, nil)
	c.Assert(names, DeepEquals, []string{
		"FixtureHelper.Test1",
		"FixtureHelper.Test2",
	})
}

// -----------------------------------------------------------------------
// Verify that verbose mode prints tests which pass as well.

func (s *RunS) TestVerboseMode(c *C) {
	helper := FixtureHelper{}
	output := String{}
	runConf := RunConf{Output: &output, Verbose: true}
	Run(c.T, &helper, &runConf)

	expected := "PASS: check_test\\.go:[0-9]+: FixtureHelper\\.Test1\t *[.0-9]+s\n" +
		"PASS: check_test\\.go:[0-9]+: FixtureHelper\\.Test2\t *[.0-9]+s\n"

	c.Assert(output.value, Matches, expected)
}

// -----------------------------------------------------------------------
// Verify the stream output mode.  In this mode there's no output caching.

type StreamHelper struct {
	l2 sync.Mutex
	l3 sync.Mutex
}

func (s *StreamHelper) SetUpSuite(c *C) {
	c.Log("0")
}

func (s *StreamHelper) Test1(c *C) {
	c.Log("1")
	s.l2.Lock()
	s.l3.Lock()
	go func() {
		s.l2.Lock() // Wait for "2".
		c.Log("3")
		s.l3.Unlock()
	}()
}

func (s *StreamHelper) Test2(c *C) {
	c.Log("2")
	s.l2.Unlock()
	s.l3.Lock() // Wait for "3".
	c.Fail()
	c.Log("4")
}

func (s *RunS) TestStreamMode(c *C) {
	helper := &StreamHelper{}
	output := String{}
	runConf := RunConf{Output: &output, Stream: true}
	Run(c.T, helper, &runConf)

	expected := "START: run_test\\.go:[0-9]+: StreamHelper\\.SetUpSuite\n0\n" +
		"PASS: run_test\\.go:[0-9]+: StreamHelper\\.SetUpSuite\t *[.0-9]+s\n\n" +
		"START: run_test\\.go:[0-9]+: StreamHelper\\.Test1\n1\n" +
		"PASS: run_test\\.go:[0-9]+: StreamHelper\\.Test1\t *[.0-9]+s\n\n" +
		"START: run_test\\.go:[0-9]+: StreamHelper\\.Test2\n2\n3\n4\n" +
		"FAIL: run_test\\.go:[0-9]+: StreamHelper\\.Test2\n\n"

	c.Assert(output.value, Matches, expected)
}

type StreamMissHelper struct{}

func (s *StreamMissHelper) SetUpSuite(c *C) {
	c.Log("0")
	c.Fail()
}

func (s *StreamMissHelper) Test1(c *C) {
	c.Log("1")
}

func (s *RunS) TestStreamModeWithMiss(c *C) {
	helper := &StreamMissHelper{}
	output := String{}
	runConf := RunConf{Output: &output, Stream: true}
	Run(c.T, helper, &runConf)

	expected := "START: run_test\\.go:[0-9]+: StreamMissHelper\\.SetUpSuite\n0\n" +
		"FAIL: run_test\\.go:[0-9]+: StreamMissHelper\\.SetUpSuite\n\n" +
		"START: run_test\\.go:[0-9]+: StreamMissHelper\\.Test1\n" +
		"MISS: run_test\\.go:[0-9]+: StreamMissHelper\\.Test1\n\n"

	c.Assert(output.value, Matches, expected)
}

// -----------------------------------------------------------------------
// Verify that that the keep work dir request indeed does so.

type WorkDirSuite struct{}

func (s *WorkDirSuite) Test(c *C) {
	c.MkDir()
}

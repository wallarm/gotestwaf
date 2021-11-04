package db

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

func TestStatistics(t *testing.T) {
	bools := []bool{false, true}

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 1000

	properties := gopter.NewProperties(parameters)

	// testPropertyNotPanics
	for _, b1 := range bools {
		for _, b2 := range bools {
			properties.Property(
				fmt.Sprintf("testPropertyNotPanics(%v, %v)-NewDBAllPassedGenerator", b1, b2),
				prop.ForAllNoShrink(
					testPropertyNotPanics,
					NewDBAllPassedGenerator(),
					BoolGenerator(b1),
					BoolGenerator(b2)))
			properties.Property(
				fmt.Sprintf("testPropertyNotPanics(%v, %v)-NewDBAllBlockedGenerator", b1, b2),
				prop.ForAllNoShrink(
					testPropertyNotPanics,
					NewDBAllBlockedGenerator(),
					BoolGenerator(b1),
					BoolGenerator(b2)))
			properties.Property(
				fmt.Sprintf("testPropertyNotPanics(%v, %v)-NewDBAllFailedGenerator", b1, b2),
				prop.ForAllNoShrink(
					testPropertyNotPanics,
					NewDBAllFailedGenerator(),
					BoolGenerator(b1),
					BoolGenerator(b2)))
			properties.Property(
				fmt.Sprintf("testPropertyNotPanics(%v, %v)-NewDBAllUnresolvedGenerator(%[1]v, %[2]v)", b1, b2),
				prop.ForAllNoShrink(
					testPropertyNotPanics,
					NewDBAllUnresolvedGenerator(b1, b2),
					BoolGenerator(b1),
					BoolGenerator(b2)))
			properties.Property(
				fmt.Sprintf("testPropertyNotPanics(%v, %v)-NewDBGenerator(%[1]v, %[2]v)", b1, b2),
				prop.ForAllNoShrink(
					testPropertyNotPanics,
					NewDBGenerator(b1, b2),
					BoolGenerator(b1),
					BoolGenerator(b2)))
		}
	}

	// testPropertyOnlyPositiveNumberValues
	for _, b1 := range bools {
		for _, b2 := range bools {
			properties.Property(
				fmt.Sprintf("testPropertyOnlyPositiveNumberValues(%v, %v)-NewDBAllPassedGenerator", b1, b2),
				prop.ForAllNoShrink(
					testPropertyOnlyPositiveNumberValues,
					NewDBAllPassedGenerator(),
					BoolGenerator(b1),
					BoolGenerator(b2)))
			properties.Property(
				fmt.Sprintf("testPropertyOnlyPositiveNumberValues(%v, %v)-NewDBAllBlockedGenerator", b1, b2),
				prop.ForAllNoShrink(
					testPropertyOnlyPositiveNumberValues,
					NewDBAllBlockedGenerator(),
					BoolGenerator(b1),
					BoolGenerator(b2)))
			properties.Property(
				fmt.Sprintf("testPropertyOnlyPositiveNumberValues(%v, %v)-NewDBAllFailedGenerator", b1, b2),
				prop.ForAllNoShrink(
					testPropertyOnlyPositiveNumberValues,
					NewDBAllFailedGenerator(),
					BoolGenerator(b1),
					BoolGenerator(b2)))
			properties.Property(
				fmt.Sprintf("testPropertyOnlyPositiveNumberValues(%v, %v)-NewDBAllUnresolvedGenerator(%[1]v, %[2]v)", b1, b2),
				prop.ForAllNoShrink(
					testPropertyOnlyPositiveNumberValues,
					NewDBAllUnresolvedGenerator(b1, b2),
					BoolGenerator(b1),
					BoolGenerator(b2)))
			properties.Property(
				fmt.Sprintf("testPropertyOnlyPositiveNumberValues(%v, %v)-NewDBGenerator(%[1]v, %[2]v)", b1, b2),
				prop.ForAllNoShrink(
					testPropertyOnlyPositiveNumberValues,
					NewDBGenerator(b1, b2),
					BoolGenerator(b1),
					BoolGenerator(b2)))
		}
	}

	// testPropertyCorrectStatValues
	for _, b1 := range bools {
		for _, b2 := range bools {
			properties.Property(
				fmt.Sprintf("testPropertyCorrectStatValues(%v, %v)-NewDBAllPassedGenerator", b1, b2),
				prop.ForAllNoShrink(
					testPropertyCorrectStatValues,
					NewDBAllPassedGenerator(),
					BoolGenerator(b1),
					BoolGenerator(b2)))
			properties.Property(
				fmt.Sprintf("testPropertyCorrectStatValues(%v, %v)-NewDBAllBlockedGenerator", b1, b2),
				prop.ForAllNoShrink(
					testPropertyCorrectStatValues,
					NewDBAllBlockedGenerator(),
					BoolGenerator(b1),
					BoolGenerator(b2)))
			properties.Property(
				fmt.Sprintf("testPropertyCorrectStatValues(%v, %v)-NewDBAllFailedGenerator", b1, b2),
				prop.ForAllNoShrink(
					testPropertyCorrectStatValues,
					NewDBAllFailedGenerator(),
					BoolGenerator(b1),
					BoolGenerator(b2)))
			properties.Property(
				fmt.Sprintf("testPropertyCorrectStatValues(%v, %v)-NewDBAllUnresolvedGenerator(%[1]v, %[2]v)", b1, b2),
				prop.ForAllNoShrink(
					testPropertyCorrectStatValues,
					NewDBAllUnresolvedGenerator(b1, b2),
					BoolGenerator(b1),
					BoolGenerator(b2)))
			properties.Property(
				fmt.Sprintf("testPropertyCorrectStatValues(%v, %v)-NewDBGenerator(%[1]v, %[2]v)", b1, b2),
				prop.ForAllNoShrink(
					testPropertyCorrectStatValues,
					NewDBGenerator(b1, b2),
					BoolGenerator(b1),
					BoolGenerator(b2)))
		}
	}

	properties.TestingRun(t)
}

func testPropertyNotPanics(db *DB, ignoreUnresolved, nonBlockedAsPassed bool) bool {
	var err interface{}

	func() {
		defer func() {
			err = recover()
		}()

		_ = db.GetStatistics(ignoreUnresolved, nonBlockedAsPassed)
	}()

	if err != nil {
		return false
	}

	return true
}

func testPropertyOnlyPositiveNumberValues(db *DB, ignoreUnresolved, nonBlockedAsPassed bool) bool {
	stat := db.GetStatistics(ignoreUnresolved, nonBlockedAsPassed)

	if stat.AllRequestsNumber < 0 ||
		stat.BlockedRequestsNumber < 0 ||
		stat.BypassedRequestsNumber < 0 ||
		stat.UnresolvedRequestsNumber < 0 ||
		stat.FailedRequestsNumber < 0 ||
		stat.ResolvedRequestsNumber < 0 ||
		stat.UnresolvedRequestsPercentage < 0 ||
		stat.ResolvedBlockedRequestsPercentage < 0 ||
		stat.ResolvedBypassedRequestsPercentage < 0 ||
		stat.FailedRequestsPercentage < 0 ||

		stat.PositiveTests.AllRequestsNumber < 0 ||
		stat.PositiveTests.BlockedRequestsNumber < 0 ||
		stat.PositiveTests.BypassedRequestsNumber < 0 ||
		stat.PositiveTests.UnresolvedRequestsNumber < 0 ||
		stat.PositiveTests.FailedRequestsNumber < 0 ||
		stat.PositiveTests.ResolvedRequestsNumber < 0 ||
		stat.PositiveTests.UnresolvedRequestsPercentage < 0 ||
		stat.PositiveTests.ResolvedFalseRequestsPercentage < 0 ||
		stat.PositiveTests.ResolvedTrueRequestsPercentage < 0 ||
		stat.PositiveTests.FailedRequestsPercentage < 0 ||

		stat.OverallRequests < 0 ||
		stat.WafScore < 0 {
		return false
	}

	summaryTablesRows := append(stat.SummaryTable, stat.PositiveTests.SummaryTable...)
	for _, row := range summaryTablesRows {
		if row.Percentage < 0 ||
			row.Sent < 0 ||
			row.Blocked < 0 ||
			row.Bypassed < 0 ||
			row.Unresolved < 0 ||
			row.Failed < 0 {
			return false
		}
	}

	return true
}

func testPropertyCorrectStatValues(db *DB, ignoreUnresolved, nonBlockedAsPassed bool) bool {
	stat := db.GetStatistics(ignoreUnresolved, nonBlockedAsPassed)

	counters := make(map[string]map[string]int)
	counters["negative"] = make(map[string]int)
	counters["positive"] = make(map[string]int)

	var wafScore float32
	var overallCompletedTestCases int
	var overallPassedRequestsPercentage float32

	for _, row := range stat.SummaryTable {
		counters["negative"]["sent"] += row.Sent
		counters["negative"]["blocked"] += row.Blocked
		counters["negative"]["bypassed"] += row.Bypassed
		counters["negative"]["unresolved"] += row.Unresolved
		counters["negative"]["failed"] += row.Failed

		totalResolvedRequests := row.Blocked + row.Bypassed
		if totalResolvedRequests != 0 {
			overallCompletedTestCases += 1
			overallPassedRequestsPercentage += calculatePercentage(row.Blocked, totalResolvedRequests)
		}
	}

	if overallCompletedTestCases != 0 {
		wafScore = overallPassedRequestsPercentage / float32(overallCompletedTestCases)
	}

	if wafScore != stat.WafScore {
		return false
	}

	counters["negative"]["all"] = counters["negative"]["blocked"] +
		counters["negative"]["bypassed"] +
		counters["negative"]["unresolved"] +
		counters["negative"]["failed"]

	counters["negative"]["resolved"] = counters["negative"]["blocked"] +
		counters["negative"]["bypassed"]

	if counters["negative"]["all"] != stat.AllRequestsNumber ||
		counters["negative"]["blocked"] != stat.BlockedRequestsNumber ||
		counters["negative"]["bypassed"] != stat.BypassedRequestsNumber ||
		counters["negative"]["unresolved"] != stat.UnresolvedRequestsNumber ||
		counters["negative"]["failed"] != stat.FailedRequestsNumber ||
		counters["negative"]["resolved"] != stat.ResolvedRequestsNumber {
		return false
	}

	for _, row := range stat.PositiveTests.SummaryTable {
		counters["positive"]["sent"] += row.Sent
		counters["positive"]["blocked"] += row.Blocked
		counters["positive"]["bypassed"] += row.Bypassed
		counters["positive"]["unresolved"] += row.Unresolved
		counters["positive"]["failed"] += row.Failed
	}

	counters["positive"]["all"] = counters["positive"]["blocked"] +
		counters["positive"]["bypassed"] +
		counters["positive"]["unresolved"] +
		counters["positive"]["failed"]

	counters["positive"]["resolved"] = counters["positive"]["blocked"] +
		counters["positive"]["bypassed"]

	if counters["positive"]["all"] != stat.PositiveTests.AllRequestsNumber ||
		counters["positive"]["blocked"] != stat.PositiveTests.BlockedRequestsNumber ||
		counters["positive"]["bypassed"] != stat.PositiveTests.BypassedRequestsNumber ||
		counters["positive"]["unresolved"] != stat.PositiveTests.UnresolvedRequestsNumber ||
		counters["positive"]["failed"] != stat.PositiveTests.FailedRequestsNumber ||
		counters["positive"]["resolved"] != stat.PositiveTests.ResolvedRequestsNumber {
		return false
	}

	overallRequests := counters["negative"]["all"] + counters["positive"]["all"]

	if overallRequests != stat.OverallRequests {
		return false
	}

	return true
}

func NewDBAllPassedGenerator() gopter.Gen {
	return gopter.DeriveGen(
		func(passedTests []Info) *DB {
			db := &DB{
				counters:      make(map[string]map[string]map[string]int),
				passedTests:   passedTests,
				numberOfTests: 0,
			}

			for _, t := range passedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				db.counters[t.Set][t.Case]["passed"] += 1
				db.numberOfTests += 1
			}
			return db
		},
		func(db *DB) []Info {
			return db.passedTests
		},
		GenInfoSlice(),
	)
}

func NewDBAllBlockedGenerator() gopter.Gen {
	return gopter.DeriveGen(
		func(blockedTests []Info) *DB {
			db := &DB{
				counters:      make(map[string]map[string]map[string]int),
				blockedTests:  blockedTests,
				numberOfTests: 0,
			}

			for _, t := range blockedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				db.counters[t.Set][t.Case]["blocked"] += 1
				db.numberOfTests += 1
			}
			return db
		},
		func(db *DB) []Info {
			return db.blockedTests
		},
		GenInfoSlice(),
	)
}

func NewDBAllUnresolvedGenerator(ignoreUnresolved, nonBlockedAsPassed bool) gopter.Gen {
	return gopter.DeriveGen(
		func(unresolvedTests []Info) *DB {
			db := &DB{
				counters:      make(map[string]map[string]map[string]int),
				naTests:       unresolvedTests,
				numberOfTests: 0,
			}

			for _, t := range unresolvedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				if (ignoreUnresolved || nonBlockedAsPassed) && !isPositiveTest(t.Set) {
					db.counters[t.Set][t.Case]["passed"]++
				} else {
					db.counters[t.Set][t.Case]["blocked"]++
				}
				db.numberOfTests += 1
			}
			return db
		},
		func(db *DB) []Info {
			return db.naTests
		},
		GenInfoSlice(),
	)
}

func NewDBAllFailedGenerator() gopter.Gen {
	return gopter.DeriveGen(
		func(failedTests []Info) *DB {
			db := &DB{
				counters:      make(map[string]map[string]map[string]int),
				failedTests:   failedTests,
				numberOfTests: 0,
			}

			for _, t := range failedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				db.counters[t.Set][t.Case]["failed"] += 1
				db.numberOfTests += 1
			}
			return db
		},
		func(db *DB) []Info {
			return db.failedTests
		},
		GenInfoSlice(),
	)
}

func NewDBGenerator(ignoreUnresolved, nonBlockedAsPassed bool) gopter.Gen {
	return gopter.DeriveGen(
		func(passedTests, blockedTests, failedTests, unresolvedTests []Info) *DB {
			db := &DB{
				counters:      make(map[string]map[string]map[string]int),
				passedTests:   passedTests,
				blockedTests:  blockedTests,
				failedTests:   failedTests,
				naTests:       unresolvedTests,
				numberOfTests: 0,
			}

			for _, t := range passedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				db.counters[t.Set][t.Case]["passed"] += 1
				db.numberOfTests += 1
			}
			for _, t := range blockedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				db.counters[t.Set][t.Case]["blocked"] += 1
				db.numberOfTests += 1
			}
			for _, t := range failedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				db.counters[t.Set][t.Case]["failed"] += 1
				db.numberOfTests += 1
			}
			for _, t := range unresolvedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				if (ignoreUnresolved || nonBlockedAsPassed) && !isPositiveTest(t.Set) {
					db.counters[t.Set][t.Case]["passed"]++
				} else {
					db.counters[t.Set][t.Case]["blocked"]++
				}
				db.numberOfTests += 1
			}
			return db
		},
		func(db *DB) ([]Info, []Info, []Info, []Info) {
			return db.passedTests, db.blockedTests, db.failedTests, db.naTests
		},
		GenInfoSlice(),
		GenInfoSlice(),
		GenInfoSlice(),
		GenInfoSlice(),
	)
}

func GenSetName() gopter.Gen {
	return func(parameters *gopter.GenParameters) *gopter.GenResult {
		setName := fmt.Sprintf("setName-%d", parameters.Rng.Intn(10))
		if rand.Intn(2) == 1 {
			setName = "false-" + setName
		}
		return gopter.NewGenResult(setName, gopter.NoShrinker)
	}
}

func GenCaseName() gopter.Gen {
	return func(parameters *gopter.GenParameters) *gopter.GenResult {
		caseName := fmt.Sprintf("caseName-%d", parameters.Rng.Intn(10))
		return gopter.NewGenResult(caseName, gopter.NoShrinker)
	}
}

func GenTestInfo() gopter.Gen {
	return gopter.DeriveGen(
		func(setName, caseName string) Info {
			return Info{
				Set:  setName,
				Case: caseName,
			}
		},
		func(i Info) (string, string) {
			return i.Set, i.Case
		},
		GenSetName(),
		GenCaseName(),
	)
}

func GenInfoSlice() gopter.Gen {
	return gen.SliceOf(GenTestInfo())
}

func BoolGenerator(b bool) gopter.Gen {
	return func(parameters *gopter.GenParameters) *gopter.GenResult {
		return gopter.NewGenResult(b, gopter.NoShrinker)
	}
}

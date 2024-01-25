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

	if stat.TruePositiveTests.AllRequestsNumber < 0 ||
		stat.TruePositiveTests.BlockedRequestsNumber < 0 ||
		stat.TruePositiveTests.BypassedRequestsNumber < 0 ||
		stat.TruePositiveTests.UnresolvedRequestsNumber < 0 ||
		stat.TruePositiveTests.FailedRequestsNumber < 0 ||
		stat.TruePositiveTests.ResolvedRequestsNumber < 0 ||
		stat.TruePositiveTests.UnresolvedRequestsPercentage < 0 ||
		stat.TruePositiveTests.ResolvedBlockedRequestsPercentage < 0 ||
		stat.TruePositiveTests.ResolvedBypassedRequestsPercentage < 0 ||
		stat.TruePositiveTests.FailedRequestsPercentage < 0 ||
		stat.TrueNegativeTests.AllRequestsNumber < 0 ||
		stat.TrueNegativeTests.BlockedRequestsNumber < 0 ||
		stat.TrueNegativeTests.BypassedRequestsNumber < 0 ||
		stat.TrueNegativeTests.UnresolvedRequestsNumber < 0 ||
		stat.TrueNegativeTests.FailedRequestsNumber < 0 ||
		stat.TrueNegativeTests.ResolvedRequestsNumber < 0 ||
		stat.TrueNegativeTests.UnresolvedRequestsPercentage < 0 ||
		stat.TrueNegativeTests.ResolvedFalseRequestsPercentage < 0 ||
		stat.TrueNegativeTests.ResolvedTrueRequestsPercentage < 0 ||
		stat.TrueNegativeTests.FailedRequestsPercentage < 0 {
		return false
	}

	summaryTablesRows := append(stat.TruePositiveTests.SummaryTable, stat.TrueNegativeTests.SummaryTable...)
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
	counters["true-positive"] = make(map[string]int)
	counters["true-negative"] = make(map[string]int)

	for _, row := range stat.TruePositiveTests.SummaryTable {
		counters["true-positive"]["sent"] += row.Sent
		counters["true-positive"]["blocked"] += row.Blocked
		counters["true-positive"]["bypassed"] += row.Bypassed
		counters["true-positive"]["unresolved"] += row.Unresolved
		counters["true-positive"]["failed"] += row.Failed
	}

	counters["true-positive"]["all"] = counters["true-positive"]["blocked"] +
		counters["true-positive"]["bypassed"] +
		counters["true-positive"]["unresolved"] +
		counters["true-positive"]["failed"]

	counters["true-positive"]["resolved"] = counters["true-positive"]["blocked"] +
		counters["true-positive"]["bypassed"]

	if counters["true-positive"]["all"] != stat.TruePositiveTests.AllRequestsNumber ||
		counters["true-positive"]["blocked"] != stat.TruePositiveTests.BlockedRequestsNumber ||
		counters["true-positive"]["bypassed"] != stat.TruePositiveTests.BypassedRequestsNumber ||
		counters["true-positive"]["unresolved"] != stat.TruePositiveTests.UnresolvedRequestsNumber ||
		counters["true-positive"]["failed"] != stat.TruePositiveTests.FailedRequestsNumber ||
		counters["true-positive"]["resolved"] != stat.TruePositiveTests.ResolvedRequestsNumber {
		return false
	}

	for _, row := range stat.TrueNegativeTests.SummaryTable {
		counters["true-negative"]["sent"] += row.Sent
		counters["true-negative"]["blocked"] += row.Blocked
		counters["true-negative"]["bypassed"] += row.Bypassed
		counters["true-negative"]["unresolved"] += row.Unresolved
		counters["true-negative"]["failed"] += row.Failed
	}

	counters["true-negative"]["all"] = counters["true-negative"]["blocked"] +
		counters["true-negative"]["bypassed"] +
		counters["true-negative"]["unresolved"] +
		counters["true-negative"]["failed"]

	counters["true-negative"]["resolved"] = counters["true-negative"]["blocked"] +
		counters["true-negative"]["bypassed"]

	if counters["true-negative"]["all"] != stat.TrueNegativeTests.AllRequestsNumber ||
		counters["true-negative"]["blocked"] != stat.TrueNegativeTests.BlockedRequestsNumber ||
		counters["true-negative"]["bypassed"] != stat.TrueNegativeTests.BypassedRequestsNumber ||
		counters["true-negative"]["unresolved"] != stat.TrueNegativeTests.UnresolvedRequestsNumber ||
		counters["true-negative"]["failed"] != stat.TrueNegativeTests.FailedRequestsNumber ||
		counters["true-negative"]["resolved"] != stat.TrueNegativeTests.ResolvedRequestsNumber {
		return false
	}

	return true
}

func NewDBAllPassedGenerator() gopter.Gen {
	return gopter.DeriveGen(
		func(passedTests []*Info) *DB {
			db := &DB{
				counters:      make(map[string]map[string]map[string]int),
				passedTests:   passedTests,
				NumberOfTests: 0,
			}

			for _, t := range passedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				db.counters[t.Set][t.Case]["passed"] += 1
				db.NumberOfTests += 1
			}
			return db
		},
		func(db *DB) []*Info {
			return db.passedTests
		},
		GenInfoSlice(),
	)
}

func NewDBAllBlockedGenerator() gopter.Gen {
	return gopter.DeriveGen(
		func(blockedTests []*Info) *DB {
			db := &DB{
				counters:      make(map[string]map[string]map[string]int),
				blockedTests:  blockedTests,
				NumberOfTests: 0,
			}

			for _, t := range blockedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				db.counters[t.Set][t.Case]["blocked"] += 1
				db.NumberOfTests += 1
			}
			return db
		},
		func(db *DB) []*Info {
			return db.blockedTests
		},
		GenInfoSlice(),
	)
}

func NewDBAllUnresolvedGenerator(ignoreUnresolved, nonBlockedAsPassed bool) gopter.Gen {
	return gopter.DeriveGen(
		func(unresolvedTests []*Info) *DB {
			db := &DB{
				counters:      make(map[string]map[string]map[string]int),
				naTests:       unresolvedTests,
				NumberOfTests: 0,
			}

			for _, t := range unresolvedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				if (ignoreUnresolved || nonBlockedAsPassed) && !isFalsePositiveTest(t.Set) {
					db.counters[t.Set][t.Case]["passed"]++
				} else {
					db.counters[t.Set][t.Case]["blocked"]++
				}
				db.NumberOfTests += 1
			}
			return db
		},
		func(db *DB) []*Info {
			return db.naTests
		},
		GenInfoSlice(),
	)
}

func NewDBAllFailedGenerator() gopter.Gen {
	return gopter.DeriveGen(
		func(failedTests []*Info) *DB {
			db := &DB{
				counters:      make(map[string]map[string]map[string]int),
				failedTests:   failedTests,
				NumberOfTests: 0,
			}

			for _, t := range failedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				db.counters[t.Set][t.Case]["failed"] += 1
				db.NumberOfTests += 1
			}
			return db
		},
		func(db *DB) []*Info {
			return db.failedTests
		},
		GenInfoSlice(),
	)
}

func NewDBGenerator(ignoreUnresolved, nonBlockedAsPassed bool) gopter.Gen {
	return gopter.DeriveGen(
		func(passedTests, blockedTests, failedTests, unresolvedTests []*Info) *DB {
			db := &DB{
				counters:      make(map[string]map[string]map[string]int),
				passedTests:   passedTests,
				blockedTests:  blockedTests,
				failedTests:   failedTests,
				naTests:       unresolvedTests,
				NumberOfTests: 0,
			}

			for _, t := range passedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				db.counters[t.Set][t.Case]["passed"] += 1
				db.NumberOfTests += 1
			}
			for _, t := range blockedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				db.counters[t.Set][t.Case]["blocked"] += 1
				db.NumberOfTests += 1
			}
			for _, t := range failedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				db.counters[t.Set][t.Case]["failed"] += 1
				db.NumberOfTests += 1
			}
			for _, t := range unresolvedTests {
				if db.counters[t.Set] == nil {
					db.counters[t.Set] = make(map[string]map[string]int)
				}
				if db.counters[t.Set][t.Case] == nil {
					db.counters[t.Set][t.Case] = make(map[string]int)
				}
				if (ignoreUnresolved || nonBlockedAsPassed) && !isFalsePositiveTest(t.Set) {
					db.counters[t.Set][t.Case]["passed"]++
				} else {
					db.counters[t.Set][t.Case]["blocked"]++
				}
				db.NumberOfTests += 1
			}
			return db
		},
		func(db *DB) ([]*Info, []*Info, []*Info, []*Info) {
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
		func(setName, caseName string) *Info {
			return &Info{
				Set:  setName,
				Case: caseName,
			}
		},
		func(i *Info) (string, string) {
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

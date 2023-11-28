package db

import (
	"encoding/csv"
	"os"
	"strconv"

	"github.com/wallarm/gotestwaf/internal/payload/encoder"
)

func (db *DB) ExportPayloads(payloadsExportFile string) error {
	csvFile, err := os.Create(payloadsExportFile)
	if err != nil {
		return err
	}
	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()

	if err := csvWriter.Write([]string{
		"Payload",
		"Check Status",
		"Response Code",
		"Placeholder",
		"Encoder",
		"Set",
		"Case",
		"Test Result",
	}); err != nil {
		return err
	}

	for _, blockedTest := range db.blockedTests {
		p := blockedTest.Payload
		e := blockedTest.Encoder
		testResult := "passed"

		ep, err := encoder.Apply(e, p)
		if err != nil {
			return err
		}

		if isFalsePositiveTest(blockedTest.Set) {
			testResult = "failed"
		}

		err = csvWriter.Write([]string{
			ep,
			"blocked",
			strconv.Itoa(blockedTest.ResponseStatusCode),
			blockedTest.Placeholder,
			blockedTest.Encoder,
			blockedTest.Set,
			blockedTest.Case,
			testResult,
		})
		if err != nil {
			return err
		}
	}

	for _, passedTest := range db.passedTests {
		p := passedTest.Payload
		e := passedTest.Encoder
		testResult := "failed"

		ep, err := encoder.Apply(e, p)
		if err != nil {
			return err
		}

		if isFalsePositiveTest(passedTest.Set) {
			testResult = "passed"
		}

		err = csvWriter.Write([]string{
			ep,
			"passed",
			strconv.Itoa(passedTest.ResponseStatusCode),
			passedTest.Placeholder,
			passedTest.Encoder,
			passedTest.Set,
			passedTest.Case,
			testResult,
		})
		if err != nil {
			return err
		}
	}

	for _, naTest := range db.naTests {
		p := naTest.Payload
		e := naTest.Encoder

		ep, err := encoder.Apply(e, p)
		if err != nil {
			return err
		}

		err = csvWriter.Write([]string{
			ep,
			"unresolved",
			strconv.Itoa(naTest.ResponseStatusCode),
			naTest.Placeholder,
			naTest.Encoder,
			naTest.Set,
			naTest.Case,
			"unknown",
		})
		if err != nil {
			return err
		}
	}

	return nil
}

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

	if err := csvWriter.Write([]string{"Payload", "Check Status", "Response Code", "Placeholder", "Encoder", "Case"}); err != nil {
		return err
	}

	for _, failedTest := range db.blockedTests {
		p := failedTest.Payload
		e := failedTest.Encoder
		ep, err := encoder.Apply(e, p)
		if err != nil {
			return err
		}
		err = csvWriter.Write([]string{ep, "blocked", strconv.Itoa(failedTest.ResponseStatusCode), failedTest.Placeholder, failedTest.Encoder, failedTest.Case})
		if err != nil {
			return err
		}
	}

	for _, passedTest := range db.passedTests {
		p := passedTest.Payload
		e := passedTest.Encoder
		ep, err := encoder.Apply(e, p)
		if err != nil {
			return err
		}
		err = csvWriter.Write([]string{ep, "passed", strconv.Itoa(passedTest.ResponseStatusCode), passedTest.Placeholder, passedTest.Encoder, passedTest.Case})
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
		err = csvWriter.Write([]string{ep, "NA", strconv.Itoa(naTest.ResponseStatusCode), naTest.Placeholder, naTest.Encoder, naTest.Case})
		if err != nil {
			return err
		}
	}

	return nil
}

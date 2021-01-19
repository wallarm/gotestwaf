package report

import (
	"encoding/csv"
	"os"

	"github.com/wallarm/gotestwaf/payload/encoder"
)

func (r *Report) ExportPayloads(payloadsExportFile string) error {
	csvFile, err := os.Create(payloadsExportFile)
	if err != nil {
		return err
	}
	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()

	for _, failedTest := range r.FailedTests {
		p := failedTest.Payload
		e := failedTest.Encoder
		ep, err := encoder.Apply(e, p)
		if err != nil {
			return err
		}
		err = csvWriter.Write([]string{ep, "failed", failedTest.Placeholder})
		if err != nil {
			return err
		}
	}

	for _, passedTest := range r.PassedTests {
		p := passedTest.Payload
		e := passedTest.Encoder
		ep, err := encoder.Apply(e, p)
		if err != nil {
			return err
		}
		err = csvWriter.Write([]string{ep, "passed", passedTest.Placeholder})
		if err != nil {
			return err
		}
	}

	for _, naTest := range r.NaTests {
		p := naTest.Payload
		e := naTest.Encoder
		ep, err := encoder.Apply(e, p)
		if err != nil {
			return err
		}
		err = csvWriter.Write([]string{ep, "NA", naTest.Placeholder})
		if err != nil {
			return err
		}
	}

	return nil
}

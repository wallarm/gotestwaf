package report

import (
	"encoding/csv"
	"gotestwaf/payload/encoder"
	"log"
	"os"
)

func (r Report) ExportPayloads(payloadsExportFile string) {

	csv_file, err := os.Create(payloadsExportFile)
	if err != nil {
		log.Fatal(err)
	}
	defer csv_file.Close()

	csv_writer := csv.NewWriter(csv_file)
	defer csv_writer.Flush()

	for _, failedTest := range r.FailedTests {
		p := failedTest.Payload
		e := failedTest.Encoder
		ep, _ := encoder.Apply(e, p)

		csv_writer.Write([]string{ep, "failed", failedTest.Placeholder})
	}

	for _, passedTest := range r.PassedTests {
		p := passedTest.Payload
		e := passedTest.Encoder
		ep, _ := encoder.Apply(e, p)

		csv_writer.Write([]string{ep, "passed", passedTest.Placeholder})
	}

	for _, naTest := range r.NaTests {
		p := naTest.Payload
		e := naTest.Encoder
		ep, _ := encoder.Apply(e, p)

		csv_writer.Write([]string{ep, "NA", naTest.Placeholder})
	}

}

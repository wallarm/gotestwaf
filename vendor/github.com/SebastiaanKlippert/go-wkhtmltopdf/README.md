[![PkgGoDev](https://pkg.go.dev/badge/github.com/SebastiaanKlippert/go-wkhtmltopdf)](https://pkg.go.dev/github.com/SebastiaanKlippert/go-wkhtmltopdf)
[![Go Report Card](https://goreportcard.com/badge/SebastiaanKlippert/go-wkhtmltopdf)](https://goreportcard.com/report/SebastiaanKlippert/go-wkhtmltopdf)
[![codebeat badge](https://codebeat.co/badges/a6bb7f66-7ae2-4de8-8b61-623ef68096c9)](https://codebeat.co/projects/github-com-sebastiaanklippert-go-wkhtmltopdf-master)
[![codecov](https://codecov.io/gh/SebastiaanKlippert/go-wkhtmltopdf/branch/master/graph/badge.svg)](https://codecov.io/gh/SebastiaanKlippert/go-wkhtmltopdf)

[![Build Status](https://github.com/SebastiaanKlippert/go-wkhtmltopdf/actions/workflows/ubuntu.yml/badge.svg?branch=master)](https://github.com/SebastiaanKlippert/go-wkhtmltopdf/actions/workflows/ubuntu.yml)
[![Build Status](https://github.com/SebastiaanKlippert/go-wkhtmltopdf/actions/workflows/macos.yml/badge.svg?branch=master)](https://github.com/SebastiaanKlippert/go-wkhtmltopdf/actions/workflows/macos.yml)

# go-wkhtmltopdf
Golang commandline wrapper for wkhtmltopdf

See http://wkhtmltopdf.org/index.html for wkhtmltopdf docs.

# What and why
We needed a way to generate PDF documents from Go. These vary from invoices with highly customizable lay-outs to reports with tables, graphs and images. In our opinion the best way to do this was by using HTML/CSS templates as source for our PDFs. Using CSS print media types and millimeters instead of pixel units we can generate very acurate PDF documents using wkhtmltopdf.

go-wkhtmltopdf is a pure Golang wrapper around the wkhtmltopdf command line utility.

It has all options typed out as struct members which makes it very easy to use if you use an IDE with
code completion and it has type safety for all options.
For example you can set general options like
```go
pdfg.Dpi.Set(600)
pdfg.NoCollate.Set(false)
pdfg.PageSize.Set(PageSizeA4)
pdfg.MarginBottom.Set(40)
``` 
The same goes for adding pages, settings page options, TOC options per page etc.

It takes care of setting the correct order of options as these can become very long with muliple pages where 
you have page and TOC options for each page.

Secondly it makes usage in server-type applications easier, every instance (PDF process) has its own output buffer 
which contains the PDF output and you can feed one input document from an io.Reader (using stdin in wkhtmltopdf).
You can combine any number of external HTML documents (HTTP(S) links) with at most one HTML document from stdin and set 
options for each input document.

Note: You can also ignore the internal buffer and let wkhtmltopdf write directly to disk if required for large files, or use the [SetOutput](https://godoc.org/github.com/SebastiaanKlippert/go-wkhtmltopdf#PDFGenerator.SetOutput) method to pass any `io.Writer`.

For us this is one of the easiest ways to generate PDF documents from Go(lang) and performance is very acceptable.

# Installation
go get or use a Go dependency manager of your liking.

```
go get -u github.com/SebastiaanKlippert/go-wkhtmltopdf
```

go-wkhtmltopdf finds the path to wkhtmltopdf by
* first looking in the current dir
* looking in the PATH and PATHEXT environment dirs
* using the WKHTMLTOPDF_PATH environment dir

If you need to set your own wkhtmltopdf path or want to change it during execution, you can call SetPath().

# Usage
See testfile ```wkhtmltopdf_test.go``` for more complex options, a common use case test is in ```simplesample_test.go``` 

```go
package wkhtmltopdf

import (
  "fmt"
  "log"
)

func ExampleNewPDFGenerator() {

  // Create new PDF generator
  pdfg, err := NewPDFGenerator()
  if err != nil {
    log.Fatal(err)
  }

  // Set global options
  pdfg.Dpi.Set(300)
  pdfg.Orientation.Set(OrientationLandscape)
  pdfg.Grayscale.Set(true)

  // Create a new input page from an URL
  page := NewPage("https://godoc.org/github.com/SebastiaanKlippert/go-wkhtmltopdf")

  // Set options for this page
  page.FooterRight.Set("[page]")
  page.FooterFontSize.Set(10)
  page.Zoom.Set(0.95)

  // Add to document
  pdfg.AddPage(page)

  // Create PDF document in internal buffer
  err = pdfg.Create()
  if err != nil {
    log.Fatal(err)
  }

  // Write buffer contents to file on disk
  err = pdfg.WriteFile("./simplesample.pdf")
  if err != nil {
    log.Fatal(err)
  }

  fmt.Println("Done")
  // Output: Done
}
```

As mentioned before, you can provide one document from stdin, this is done by using a [PageReader](https://godoc.org/github.com/SebastiaanKlippert/go-wkhtmltopdf#PageReader "GoDoc") object as input to AddPage. This is best constructed with  [NewPageReader](https://godoc.org/github.com/SebastiaanKlippert/go-wkhtmltopdf#NewPageReader "GoDoc") and will accept any io.Reader so this can be used with files from disk (os.File) or memory (bytes.Buffer) etc.  
A simple example snippet:
```go
html := "<html>Hi</html>"
pdfgen.AddPage(NewPageReader(strings.NewReader(html)))
```

# Saving to and loading from JSON

The package now has the possibility to save the PDF Generator object as JSON and to create
a new PDF Generator from a JSON file.
All options and pages are saved in JSON, pages added using NewPageReader are read to memory before saving and then saved as Base64 encoded strings
in the JSON file.

This is useful to prepare a PDF file and generate the actual PDF elsewhere, for example on AWS Lambda.
To create PDF Generator on the client, where wkhtmltopdf might not be present, function `NewPDFPreparer` can be used.

Use `NewPDFPreparer` to create a PDF Generator object on the client and `NewPDFGeneratorFromJSON` to reconstruct it on the server.

```go 
// Client code
pdfg := NewPDFPreparer()
htmlfile, err := ioutil.ReadFile("testdata/htmlsimple.html")
if err != nil {
  log.Fatal(err)
}
    
pdfg.AddPage(NewPageReader(bytes.NewReader(htmlfile)))
pdfg.Dpi.Set(600)
    
// The contents of htmlsimple.html are saved as base64 string in the JSON file
jb, err := pdfg.ToJSON()
if err != nil {
  log.Fatal(err)
}
    
// Server code
pdfgFromJSON, err := NewPDFGeneratorFromJSON(bytes.NewReader(jb))
if err != nil {
  log.Fatal(err)
}
    
err = pdfgFromJSON.Create()
if err != nil {
  log.Fatal(err)
}    
```

For an example of running this in AWS Lambda see https://github.com/SebastiaanKlippert/go-wkhtmltopdf-lambda

# Speed 
The speed if pretty much determined by wkhtmltopdf itself, or if you use external source URLs, the time it takes to get and render the source HTML.

The go wrapper time is negligible with around 0.04ms for parsing an above average number of commandline options.

Benchmarks are included.

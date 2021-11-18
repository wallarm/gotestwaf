// Package wkhtmltopdf contains wrappers around the wkhtmltopdf commandline tool
package wkhtmltopdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

//the cached mutexed path as used by findPath()
type stringStore struct {
	val string
	sync.Mutex
}

func (ss *stringStore) Get() string {
	ss.Lock()
	defer ss.Unlock()
	return ss.val
}

func (ss *stringStore) Set(s string) {
	ss.Lock()
	ss.val = s
	ss.Unlock()
}

var binPath stringStore

// SetPath sets the path to wkhtmltopdf
func SetPath(path string) {
	binPath.Set(path)
}

// GetPath gets the path to wkhtmltopdf
func GetPath() string {
	return binPath.Get()
}

// Page is the input struct for each page
type Page struct {
	Input string
	PageOptions
}

// InputFile returns the input string and is part of the page interface
func (p *Page) InputFile() string {
	return p.Input
}

// Args returns the argument slice and is part of the page interface
func (p *Page) Args() []string {
	return p.PageOptions.Args()
}

// Reader returns the io.Reader and is part of the page interface
func (p *Page) Reader() io.Reader {
	return nil
}

// NewPage creates a new input page from a local or web resource (filepath or URL)
func NewPage(input string) *Page {
	return &Page{
		Input:       input,
		PageOptions: NewPageOptions(),
	}
}

// PageReader is one input page (a HTML document) that is read from an io.Reader
// You can add only one Page from a reader
type PageReader struct {
	Input io.Reader
	PageOptions
}

// InputFile returns the input string and is part of the page interface
func (pr *PageReader) InputFile() string {
	return "-"
}

// Args returns the argument slice and is part of the page interface
func (pr *PageReader) Args() []string {
	return pr.PageOptions.Args()
}

//Reader returns the io.Reader and is part of the page interface
func (pr *PageReader) Reader() io.Reader {
	return pr.Input
}

// NewPageReader creates a new PageReader from an io.Reader
func NewPageReader(input io.Reader) *PageReader {
	return &PageReader{
		Input:       input,
		PageOptions: NewPageOptions(),
	}
}

// PageProvider is the interface which provides a single input page.
// Implemented by Page and PageReader.
type PageProvider interface {
	Args() []string
	InputFile() string
	Reader() io.Reader
}

// PageOptions are options for each input page
type PageOptions struct {
	pageOptions
	headerAndFooterOptions
}

// Args returns the argument slice
func (po *PageOptions) Args() []string {
	return append(append([]string{}, po.pageOptions.Args()...), po.headerAndFooterOptions.Args()...)
}

// NewPageOptions returns a new PageOptions struct with all options
func NewPageOptions() PageOptions {
	return PageOptions{
		pageOptions:            newPageOptions(),
		headerAndFooterOptions: newHeaderAndFooterOptions(),
	}
}

// cover page
type cover struct {
	Input string
	pageOptions
}

// table of contents
type toc struct {
	Include bool
	allTocOptions
}

type allTocOptions struct {
	pageOptions
	tocOptions
	headerAndFooterOptions
}

// PDFGenerator is the main wkhtmltopdf struct, always use NewPDFGenerator to obtain a new PDFGenerator struct
type PDFGenerator struct {
	globalOptions
	outlineOptions

	Cover      cover
	TOC        toc
	OutputFile string //filename to write to, default empty (writes to internal buffer)

	binPath   string
	outbuf    bytes.Buffer
	outWriter io.Writer
	stdErr    io.Writer
	pages     []PageProvider
}

//Args returns the commandline arguments as a string slice
func (pdfg *PDFGenerator) Args() []string {
	args := append([]string{}, pdfg.globalOptions.Args()...)
	args = append(args, pdfg.outlineOptions.Args()...)
	if pdfg.Cover.Input != "" {
		args = append(args, "cover")
		args = append(args, pdfg.Cover.Input)
		args = append(args, pdfg.Cover.pageOptions.Args()...)
	}
	if pdfg.TOC.Include {
		args = append(args, "toc")
		args = append(args, pdfg.TOC.pageOptions.Args()...)
		args = append(args, pdfg.TOC.tocOptions.Args()...)
		args = append(args, pdfg.TOC.headerAndFooterOptions.Args()...)
	}
	for _, page := range pdfg.pages {
		args = append(args, "page")
		args = append(args, page.InputFile())
		args = append(args, page.Args()...)
	}
	if pdfg.OutputFile != "" {
		args = append(args, pdfg.OutputFile)
	} else {
		args = append(args, "-")
	}
	return args
}

// ArgString returns Args as a single string
func (pdfg *PDFGenerator) ArgString() string {
	return strings.Join(pdfg.Args(), " ")
}

// AddPage adds a new input page to the document.
// A page is an input HTML page, it can span multiple pages in the output document.
// It is a Page when read from file or URL or a PageReader when read from memory.
func (pdfg *PDFGenerator) AddPage(p PageProvider) {
	pdfg.pages = append(pdfg.pages, p)
}

// SetPages resets all pages
func (pdfg *PDFGenerator) SetPages(p []PageProvider) {
	pdfg.pages = p
}

// ResetPages drops all pages previously added by AddPage or SetPages.
// This allows reuse of current instance of PDFGenerator with all of it's configuration preserved.
func (pdfg *PDFGenerator) ResetPages() {
	pdfg.pages = []PageProvider{}
}

// Buffer returns the embedded output buffer used if OutputFile is empty
func (pdfg *PDFGenerator) Buffer() *bytes.Buffer {
	return &pdfg.outbuf
}

// Bytes returns the output byte slice from the output buffer used if OutputFile is empty
func (pdfg *PDFGenerator) Bytes() []byte {
	return pdfg.outbuf.Bytes()
}

// SetOutput sets the output to write the PDF to, when this method is called, the internal buffer will not be used,
// so the Bytes(), Buffer() and WriteFile() methods will not work.
func (pdfg *PDFGenerator) SetOutput(w io.Writer) {
	pdfg.outWriter = w
}

// SetStderr sets the output writer for Stderr when running the wkhtmltopdf command. You only need to call this when you
// want to print the output of wkhtmltopdf (like the progress messages in verbose mode). If not called, or if w is nil, the
// output of Stderr is kept in an internal buffer and returned as error message if there was an error when calling wkhtmltopdf.
func (pdfg *PDFGenerator) SetStderr(w io.Writer) {
	pdfg.stdErr = w
}

// WriteFile writes the contents of the output buffer to a file
func (pdfg *PDFGenerator) WriteFile(filename string) error {
	return ioutil.WriteFile(filename, pdfg.Bytes(), 0666)
}

//findPath finds the path to wkhtmltopdf by
//- first looking in the current dir
//- looking in the PATH and PATHEXT environment dirs
//- using the WKHTMLTOPDF_PATH environment dir
//The path is cached, meaning you can not change the location of wkhtmltopdf in
//a running program once it has been found
func (pdfg *PDFGenerator) findPath() error {
	const exe = "wkhtmltopdf"
	pdfg.binPath = GetPath()
	if pdfg.binPath != "" {
		// wkhtmltopdf has already already found, return
		return nil
	}
	exeDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}
	path, err := exec.LookPath(filepath.Join(exeDir, exe))
	if err == nil && path != "" {
		binPath.Set(path)
		pdfg.binPath = path
		return nil
	}
	path, err = exec.LookPath(exe)
	if err == nil && path != "" {
		binPath.Set(path)
		pdfg.binPath = path
		return nil
	}
	dir := os.Getenv("WKHTMLTOPDF_PATH")
	if dir == "" {
		return fmt.Errorf("%s not found", exe)
	}
	path, err = exec.LookPath(filepath.Join(dir, exe))
	if err == nil && path != "" {
		binPath.Set(path)
		pdfg.binPath = path
		return nil
	}
	return fmt.Errorf("%s not found", exe)
}

// Create creates the PDF document and stores it in the internal buffer if no error is returned
func (pdfg *PDFGenerator) Create() error {
	return pdfg.run(context.Background())
}

// CreateContext is Create with a context passed to exec.CommandContext when calling wkhtmltopdf
func (pdfg *PDFGenerator) CreateContext(ctx context.Context) error {
	return pdfg.run(ctx)
}

func (pdfg *PDFGenerator) run(ctx context.Context) error {
	// create command
	cmd := exec.CommandContext(ctx, pdfg.binPath, pdfg.Args()...)

	// set stderr to the provided writer, or create a new buffer
	var errBuf *bytes.Buffer
	cmd.Stderr = pdfg.stdErr
	if cmd.Stderr == nil {
		errBuf = new(bytes.Buffer)
		cmd.Stderr = errBuf
	}

	// set output to the desired writer or the internal buffer
	if pdfg.outWriter != nil {
		cmd.Stdout = pdfg.outWriter
	} else {
		cmd.Stdout = &pdfg.outbuf
	}

	// if there is a pageReader page (from Stdin) we set Stdin to that reader
	for _, page := range pdfg.pages {
		if page.Reader() != nil {
			cmd.Stdin = page.Reader()
			break
		}
	}

	// run cmd to create the PDF
	err := cmd.Run()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		// on an error, return the contents of Stderr if it was our own buffer
		// if Stderr was set to a custom writer, just return err
		if errBuf != nil {
			if errStr := errBuf.String(); strings.TrimSpace(errStr) != "" {
				return errors.New(errStr)
			}
		}
		return err
	}
	return nil
}

// NewPDFGenerator returns a new PDFGenerator struct with all options created and
// checks if wkhtmltopdf can be found on the system
func NewPDFGenerator() (*PDFGenerator, error) {
	pdfg := NewPDFPreparer()
	return pdfg, pdfg.findPath()
}

// NewPDFPreparer returns a PDFGenerator object without looking for the wkhtmltopdf executable file.
// This is useful to prepare a PDF file that is generated elsewhere and you just want to save the config as JSON.
// Note that Create() can not be called on this object unless you call SetPath yourself.
func NewPDFPreparer() *PDFGenerator {
	return &PDFGenerator{
		globalOptions:  newGlobalOptions(),
		outlineOptions: newOutlineOptions(),
		Cover: cover{
			pageOptions: newPageOptions(),
		},
		TOC: toc{
			allTocOptions: allTocOptions{
				tocOptions:  newTocOptions(),
				pageOptions: newPageOptions(),
				headerAndFooterOptions: newHeaderAndFooterOptions(),
			},
		},
	}
}

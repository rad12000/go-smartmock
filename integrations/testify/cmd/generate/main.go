/*
Generate generates generic mock wrapper functions, based on the flags provided.

Usage:

	generate [flags]

The flags are:

	--minArg
	    The minimum argument to start at. E.g. if min arg is 2, then generated functions would
		start at 2 and create functions for 2, 3, 4, etc. arguments.
	--minReturn
		The minimum return to start at. E.g. if min return is 2, then generated functions would
		start at 2 and create functions for 2, 3, 4, etc. return values.
	--maxArg
	    The maximum argument to end at. E.g. if --maxArg=5 and --minArg=2, then generated functions would
		end at 5 and create functions for 2, 3, 4, and 5 arguments.
	--maxReturn
		The max return to end at. E.g. if --maxReturn=5 and --minReturn=2, then generated functions would
		end at 5 and create functions for 2, 3, 4, and 5 return values.
	--parallel
		If provided, then the functions will be generated in parallel. This is useful for generating a large
		number of functions, but will cause the output to be non-deterministic.
*/
package main

import (
	"bytes"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"
)

var (
	//go:embed template
	tmpl      string
	parseTmpl = template.Must(template.New("generate").Funcs(funcMap).Parse(tmpl))
	funcMap   = template.FuncMap{
		"toUpper":  strings.ToUpper,
		"toLower":  strings.ToLower,
		"minusOne": func(n int) int { return n - 1 },
	}

	minArg    uint
	minReturn uint
	maxArg    uint
	maxReturn uint
	parallel  bool
	bufPool   = sync.Pool{New: func() any {
		return &bytes.Buffer{}
	}}
)

type templateData struct {
	PackageName string
	IncludeFmt  bool
	SourceFile  string
	Funcs       []mockFn
}

type mockFn struct {
	// GenericArgs is a combo of GenericParams and GenericReturns.
	GenericArgs    []string
	GenericParams  []string
	GenericReturns []string
}

func main() {
	t := time.Now()
	defer func() {
		fmt.Println("Took", time.Since(t).Milliseconds(), "ms")
	}()
	flag.UintVar(&minArg, "minArg", 0, "The minimum number of arguments to generate.")
	flag.UintVar(&minReturn, "minReturn", 0, "The minimum number of return arguments to generate.")
	flag.UintVar(&maxArg, "maxArg", 0, "The max number of arguments to generate.")
	flag.UintVar(&maxReturn, "maxReturn", 0, "The max number of return arguments to generate.")
	flag.BoolVar(&parallel, "parallel", false, "Whether or not to generate functions in parallel.")
	flag.Parse()

	file, err := os.OpenFile(fmt.Sprintf("generated_%s", os.Getenv("GOFILE")), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	tmplData := templateData{
		PackageName: os.Getenv("GOPACKAGE"),
		IncludeFmt:  maxArg > 0 && minArg <= maxArg,
		SourceFile:  fmt.Sprintf("%s.%s", os.Getenv("GOPACKAGE"), os.Getenv("GOFILE")),
	}

	for i := minArg; i <= maxArg; i++ {
		for j := minReturn; j <= maxReturn; j++ {
			var fn mockFn
			var n uint
			for n = 0; n < i; n++ {
				fn.GenericParams = append(fn.GenericParams, fmt.Sprintf("P%d", n))
			}

			for n = 0; n < j; n++ {
				fn.GenericReturns = append(fn.GenericReturns, fmt.Sprintf("R%d", n))
			}

			fn.GenericArgs = append(fn.GenericParams, fn.GenericReturns...)
			tmplData.Funcs = append(tmplData.Funcs, fn)
		}
	}

	w := newThreadSafeWriter(file)
	err = parseTmpl.ExecuteTemplate(w, "generate", tmplData)
	_, err2 := w.flush()
	if err := errors.Join(err, err2); err != nil {
		panic(err)
	}

	numCPU := 1
	if parallel {
		numCPU := runtime.NumCPU() - 1
		if numCPU < 1 {
			numCPU = 1
		}
	}

	ch := make(chan struct{}, numCPU)
	var wg sync.WaitGroup
	for _, fn := range tmplData.Funcs {
		ch <- struct{}{}
		wg.Add(1)
		go func(fn mockFn) {
			defer func() {
				wg.Done()
				<-ch
			}()

			w := w.clone()
			err := parseTmpl.ExecuteTemplate(w, "smartFunc", fn)
			_, err2 := w.flush()
			if err := errors.Join(err, err2); err != nil {
				panic(err)
			}
		}(fn)
	}
	wg.Wait()
	fmt.Println("Created", (maxArg+1)*(maxReturn+1)*2, "functions")
}

func newThreadSafeWriter(w io.Writer) threadSafeWriter {
	return threadSafeWriter{
		w:   w,
		mu:  &sync.Mutex{},
		buf: bufPool.Get().(*bytes.Buffer),
	}
}

type threadSafeWriter struct {
	w   io.Writer
	mu  *sync.Mutex
	buf *bytes.Buffer
}

func (w threadSafeWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w threadSafeWriter) clone() threadSafeWriter {
	w.buf = bufPool.Get().(*bytes.Buffer)
	return w
}

func (w threadSafeWriter) flush() (int64, error) {
	defer func() {
		w.buf.Reset()
		bufPool.Put(w.buf)
	}()
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.WriteTo(w.w)
}

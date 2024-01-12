/*
Generate generates generic mock wrapper functions, based on the flags provided.

Usage:

	gofmt [flags] [path ...]

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
*/
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"
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
	flag.UintVar(&minArg, "minArg", 0, "The minimum number of arguments to generate.")
	flag.UintVar(&minReturn, "minReturn", 0, "The minimum number of return arguments to generate.")
	flag.UintVar(&maxArg, "maxArg", 0, "The max number of arguments to generate.")
	flag.UintVar(&maxReturn, "maxReturn", 0, "The max number of return arguments to generate.")
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

	err = parseTmpl.ExecuteTemplate(file, "generate", tmplData)
	if err != nil {
		panic(err)
	}
}

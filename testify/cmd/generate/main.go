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

	minArg    int
	minReturn int
	maxArg    int
	maxReturn int
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
	flag.IntVar(&minArg, "minArg", 0, "The minimum number of arguments to generate.")
	flag.IntVar(&minReturn, "minReturn", 0, "The minimum number of return arguments to generate.")
	flag.IntVar(&maxArg, "maxArg", 0, "The max number of arguments to generate.")
	flag.IntVar(&maxReturn, "maxReturn", 0, "The max number of return arguments to generate.")
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
			for n := 0; n < i; n++ {
				fn.GenericParams = append(fn.GenericParams, fmt.Sprintf("P%d", n))
			}

			for n := 0; n < j; n++ {
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

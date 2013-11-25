package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/naoina/kocha"
	"github.com/naoina/kocha/kocha/generator"
	"os"
	"text/template"
)

// newCommand implements `command` interface for `generate` command.
type generateCommand struct {
	flag *flag.FlagSet
}

// Name returns name of `generate` command.
func (c *generateCommand) Name() string {
	return "generate"
}

// Alias returns alias of `generate` command.
func (c *generateCommand) Alias() string {
	return "g"
}

// Short returns short description for help.
func (c *generateCommand) Short() string {
	return "generate files"
}

// Usage returns usage of `generate` command.
func (c *generateCommand) Usage() string {
	var buf bytes.Buffer
	template.Must(template.New("usage").Parse(`%s GENERATOR [args]

Generators:
{{range $name, $ := .}}
    {{$name|printf "%-6s"}}{{end}}
`)).Execute(&buf, generator.Generators)
	return fmt.Sprintf(buf.String(), c.Name())
}

func (c *generateCommand) DefineFlags(fs *flag.FlagSet) {
	c.flag = fs
}

// Run execute the process for `generate` command.
func (c *generateCommand) Run() {
	generatorName := c.flag.Arg(0)
	if generatorName == "" {
		kocha.PanicOnError(c, "abort: no GENERATOR given")
	}
	generator := generator.Get(generatorName)
	if generator == nil {
		kocha.PanicOnError(c, "abort: could not find generator: %v", generatorName)
	}
	flagSet := flag.NewFlagSet(generatorName, flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s %s %s\n", os.Args[0], c.Name(), generator.Usage())
	}
	defer func() {
		if err := recover(); err != nil {
			if err, ok := err.(kocha.Error); ok {
				fmt.Fprintln(os.Stderr, err.Message)
				fmt.Fprintf(os.Stderr, "usage: %s %s %s\n", os.Args[0], c.Name(), err.Usager.Usage())
				os.Exit(1)
			}
			panic(err)
		}
	}()
	generator.DefineFlags(flagSet)
	flagSet.Parse(c.flag.Args()[1:])
	generator.Generate()
}

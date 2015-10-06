package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"

	"github.com/sdboyer/memoize/parse"
	"github.com/tinylib/msgp/printer"
	"github.com/ttacon/chalk"
)

var (
	out  = flag.String("o", "", "output file")
	file = flag.String("file", "", "input file")
	//tests      = flag.Bool("tests", true, "create tests and benchmarks")
)

func main() {
	flag.Parse()

	// GOFILE is set by go generate, use it unless one was provided
	if *file == "" {
		*file = os.Getenv("GOFILE")
		if *file == "" {
			fmt.Println(chalk.Red.Color("No file to parse."))
			os.Exit(1)
		}
	}

	if err := Run(&build.Default, *file); err != nil {
		fmt.Println(chalk.Red.Color(err.Error()))
		os.Exit(1)
	}
}

func Run(ctxt *build.Context, gofile string) error {
	fmt.Println(chalk.Magenta.Color("======== Memoization Code Generator ======="))
	fmt.Printf(chalk.Magenta.Color(">>> Input: \"%s\"\n"), gofile)
	fs, err := parse.File(gofile)
	if err != nil {
		return err
	}

	if len(fs.Identities) == 0 {
		fmt.Println(chalk.Magenta.Color("No types requiring code generation were found!"))
		return nil
	}

	return printer.PrintFile(newFilename(gofile, fs.Package), fs, mode)
}

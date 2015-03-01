# flags

``flags`` is a command line options parser aimed at providing more options than
the standard `flag` Go package. It was inspired after the `argparse` Python module, and allows the following (among other things):

* short and long flag ("-i" or "--integer")
* open files automatically with the given filepath
* automatic help message generation
* limit the values of the flags to a predefined set of values
* positional arguments
* capture values to slices
* and more!

## Technical notes

First off, this is a work in progress which I have been working on for several
months, on and off with several weeks between the times I actually worked on
the code. It might not do everything that `argparse` allows Python coders to
do, but it is still an improvement to the standard `flag` package.

I did not base the implementation of ``flags`` on the standard `flag` package.
The default way of getting arguments from the command line is very simple, I
needed to have more control over how the flags were parsed, so I just rolled
my own implementation.

There is no default format for the flags you want to use in your command line
application, unlike with the standard `flag` package. You can use whatever
string you want as a flag, and are not limited to the "-flagname" format. The
``flags`` package also allows you to use the same usual different ways of
setting values:

* ``-flag 123``
* ``--flag 123``
* ``-f 123``
* ``--flag=123``
* ``flag=123``

The above list shows what value formats it is possible to parse with ``flags``,
it is up to you which ones will be available when you call the ``*Var()``
functions (c.f example).

## Example
```
/*
 * example.go for flags
 * by lenormf
 */

package main

import (
	"fmt"
	"github.com/lenormf/flags"
	"os"
	"strings"
)

// This structure will hold all the values parsed from the CLI
type CliArguments struct {
	Integer       int
	FileOutput    *os.File
	Sentence      []string
	PrintSentence bool
}

func main() {
	var err error
	cli := &CliArguments{}
	parser := flags.NewArgumentsParser(os.Args[0], "Description of my program")

	// Declare a mandatory integer flag whose value is [1-4]
	err = parser.IntVar(&cli.Integer, "--integer", "An integer", &flags.IntVarOptions{
		ShortFlag: "-i",
		Required:  true,
		Choices:   []int{1, 2, 3, 4},
	})
	if err != nil {
		// OnParsingError is the default function that will be called when an
		// error is triggered, it prints the help message of the parser
		// (generated automatically) and issues an os.Exit(1) call
		flags.OnParsingError(parser, err)
	}

	// Use the value of the required "-o"/"--output" flag as a filepath,
	// and open a file at that location in write mode
	err = parser.FileVar(&cli.FileOutput, "--output", "Path to the output file", &flags.FileVarOptions{
		ShortFlag: "-o",
		Mode:      "w",
		Perms:     0640,
		Required:  true,
	})
	// The parser keeps track of all the files that were open by it, it can
	// also bulk free all open file descriptors
	defer parser.CloseAllOpenFiles()
	if err != nil {
		flags.OnParsingError(parser, err)
	}

	// Grab all the remaining unparsed values off of the CLI,
	// and save them in a slice
	parser.StringVar(&cli.Sentence, "sentence", "A few words", &flags.StringVarOptions{
		NArgs: -1,
	})

	parser.BoolVar(&cli.PrintSentence, "--print", "Print the sentence on the standard output", &flags.BoolVarOptions{
		ShortFlag:    "-p",
		Default:      false,
		ValueOnExist: true,
	})

	remaining_args, err := parser.Parse(os.Args[1:])
	if err != nil {
		flags.OnParsingError(parser, err)
	}
	if len(remaining_args) > 0 {
		fmt.Printf("Unparsed command line arguments: %v\n", remaining_args)
	}

	sentence := strings.Join(cli.Sentence, " ")

	cli.FileOutput.WriteString(sentence)
	cli.FileOutput.Close()

	if cli.PrintSentence {
		if len(cli.Sentence) > 0 {
			fmt.Printf("%s\n", sentence)
		} else {
			fmt.Printf("No sentence was passed !")
		}
	}
}
```

## License

The entire code within this repository is placed under the MIT license.

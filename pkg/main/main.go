package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"

	"github.com/claudiolor/textfsmgo/pkg/textfsmgo"
	"github.com/claudiolor/textfsmgo/pkg/utils"
)

func showError(err error, ecode int) {
	fmt.Println(err.Error())
	os.Exit(ecode)
}

func setupFlagUsage() {
	flag.Usage = func() {
		usage_str := fmt.Sprintf("Usage: %s FILE_NAME TEMPLATE_FILE [..args]", os.Args[0])
		fmt.Println(usage_str)
		fmt.Println("Args:")
		flag.PrintDefaults()
	}
}

func showUsage() {
	flag.Usage()
	os.Exit(1)
}

func main() {
	out_file := flag.String("o", "", "Write the result in an output file instead of stdout")
	intend := flag.Bool("i", false, "Show the json output with indentation")
	setupFlagUsage()
	flag.Parse()

	if flag.NArg() != 2 {
		showUsage()
	}
	in_file := flag.Arg(0)
	tmpl_file := flag.Arg(1)
	parser, err := textfsmgo.NewTextFSMParser(tmpl_file)
	if err != nil {
		showError(err, 1)
	}

	input_str, err := os.ReadFile(in_file)
	if err != nil {
		showError(err, 1)
	}

	res, err := parser.ParseTextToDicts(string(input_str))
	if err != nil {
		showError(err, 1)
	}

	jsonRes, err := utils.ConvertResToJson(&res, *intend)
	if err != nil {
		showError(err, 1)
	}

	// Check whether we should print to stdout or produce a file
	if *out_file == "" {
		fmt.Println(string(jsonRes))
	} else {
		err := os.WriteFile(*out_file, jsonRes, fs.FileMode(0664))
		if err != nil {
			showError(err, 1)
		}
		fmt.Printf("Json file %s written!\n", *out_file)
	}
}

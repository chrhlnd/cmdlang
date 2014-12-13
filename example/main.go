package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/chrhlnd/cmdlang"
)

var DATA = `
this is a  command
	,that continues to here

literals 'are in quotes' or these "quotes"

this 'is a command' that (some sub command for this param) calls sub commands

someaction
	,(depends on sub action)
	,(and this sub action)

# this is an eol commment

#(
	This is a block comment?

	)#

	#( something )#
	
`

func main() {
	flag.Parse()

	for _, f := range flag.Args() {
		fmt.Print(">>> ---------------- ")
		fmt.Println(f)

		file, err := os.Open(f)
		if err != nil {
			fmt.Print("ERR: ")
			fmt.Println(err)
		} else {
			scanner := cmdlang.NewScanner(file)

			var tok cmdlang.TokInfo

			for tok = scanner.Scan(); tok.Token != cmdlang.TOK_EOF; tok = scanner.Scan() {
				fmt.Printf("%v\n", tok)
			}
			fmt.Printf("%v\n", tok)

			file.Close()
		}

		fmt.Println(f)
		fmt.Println("<<< ---------------- ")
	}
}

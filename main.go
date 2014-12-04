package main

import (
	"bytes"
	"fmt"
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
	buf := bytes.NewBufferString(DATA)

	scanner := NewScanner(buf)

	var tok TokInfo

	for tok = scanner.Scan(); tok.Token != TOK_EOF; tok = scanner.Scan() {
		fmt.Printf("%v\n", tok)
	}
	fmt.Printf("%v\n", tok)

	fmt.Println(DATA)

}

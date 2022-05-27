package main

import (
	"fmt"
	"os"

	"github.com/mitchellh/colorstring"
)

func main() {
	fmt.Fprintf(os.Stderr, colorstring.Color("[red]not implemented"))
	os.Exit(1)
}

package main

import (
	"fmt"
	"os"

	"github.com/mitchellh/colorstring"
)

func main() {
	fmt.Fprint(os.Stderr, colorstring.Color("[red]not implemented\n"))
	os.Exit(1)
}

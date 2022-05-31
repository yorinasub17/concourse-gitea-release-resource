package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mitchellh/colorstring"
)

func InputRequest(request interface{}) {
	if err := json.NewDecoder(os.Stdin).Decode(request); err != nil {
		fmt.Fprintf(os.Stderr, colorstring.Color("[red]error reading request form stdin: %s\n"), err)
		os.Exit(1)
	}
}

func OutputResponse(response interface{}) {
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, colorstring.Color("[red]error writing response to stdout: %s\n"), err)
		os.Exit(1)
	}
}

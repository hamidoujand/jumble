package main

import (
	"fmt"
	"os"

	"github.com/hamidoujand/jumble/cmd/admin/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

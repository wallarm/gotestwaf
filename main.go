package main

import (
	"os"

	"github.com/wallarm/gotestwaf/cmd"
)

func main() {
	os.Exit(cmd.Run())
}

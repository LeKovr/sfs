//+build !test

// This file holds code which does not covered by tests

package main

import (
	"log"
	"os"
)

// Actual version value will be set at build time
var version = "0.0-dev"

func main() {
	log.Printf("sfs %s. Sample FileServer", version)
	run(os.Exit)
}

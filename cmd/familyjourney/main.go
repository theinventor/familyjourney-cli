// Command familyjourney is the official CLI for FamilyJourney's parent API.
package main

import (
	"fmt"
	"os"

	"github.com/theinventor/familyjourney-cli/internal/cmd"
	"github.com/theinventor/familyjourney-cli/internal/exitcode"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "familyjourney:", err)
		os.Exit(exitcode.ExitCodeFor(err))
	}
}

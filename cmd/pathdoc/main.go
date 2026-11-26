// pathdoc finds where a non-ASCII path will break a developer's build on
// Windows, and fixes it reversibly.
package main

import (
	"os"

	"github.com/Fd3os/pathdoc/internal/cli"
)

func main() { os.Exit(cli.Run(os.Args[1:], cli.Real())) }

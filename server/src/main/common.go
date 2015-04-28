package "main"

import (
  "log"
  "os"
)

func assert(condition bool, callSite string) {
  if !condition {
    log.Printf("Assertion failed in %s, abort.\n", callSite)
    os.Exit(1)
  }
}
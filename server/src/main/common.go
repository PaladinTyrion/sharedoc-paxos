package "main"

import (
  "log"
  "os"
)

const (
  NoOp     = iota
  InsertOp
  DeleteOp
)

type Op struct {
  Cid  int
  Seq  int
  Rev  uint64
  Opty int
  Pos  uint64
  Char string
}

func assert(condition bool, callSite string) {
  if !condition {
    log.Printf("Assertion failed in %s, abort.\n", callSite)
    os.Exit(1)
  }
}

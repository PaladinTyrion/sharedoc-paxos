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

func port(host int) string {
  s := fmt.Sprintf("/var/tmp/824proj-%v/", os.Getuid())
  os.Mkdir(s, 0777)
  s += fmt.Sprintf("px-%v-%v", os.Getpid(), host)
  return s
}

func assert(condition bool, callSite string) {
  if !condition {
    log.Printf("Assertion failed in %s, abort.\n", callSite)
    os.Exit(1)
  }
}
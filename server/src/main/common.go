package main

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
  ID       int64
  Version  uint64
  Type     int
  Position uint64
  Value    string
}

func assert(condition bool, callSite string) {
  if !condition {
    log.Printf("Assertion failed in %s, abort.\n", callSite)
    os.Exit(1)
  }
}

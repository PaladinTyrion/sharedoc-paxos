package "main"

import (
  "sync"
  "paxos"
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

type EPServer struct {
  mu   sync.Mutex
  px   *paxos.Paxos
  pads map[int64]PadManager // pad id -> the actual etherpad manager
}

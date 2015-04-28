package "main"

import "paxos"

type Op struct {
  Cid  int
  Seq  int
  Rev  uint64
  Opty int
  Pos  uint64
  Char string
}

type EPServer struct {
  px *paxos.Paxos
}

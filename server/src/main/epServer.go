package main

import (
  "log"
  "sync"
  "crypto/rand"
  "paxos"
  "math/big"
  "encoding/gob"
  "github.com/googollee/go-socket.io"
)

type EPServer struct {
  mu          sync.Mutex
  sio         *socketio.Server
  px          *paxos.Paxos
  skts        map[string]string     // socket id -> pad id
                                   // live session information
  pads        map[string]*PadManager // pad id -> the actual etherpad
                                   // manager, paxos-agreed state
  commitPoint int
}

func nrand() int64 {
  max := big.NewInt(int64(1) << 62)
  bigx, _ := rand.Int(rand.Reader, max)
  x := bigx.Int64()
  return x
}

func (es *EPServer) getPadById(padId string) *PadManager {
  es.mu.Lock()
  defer es.mu.Unlock()

  pm, ok := es.pads[padId]
  if !ok {
    pm = NewPadManager(padId)
    es.pads[padId] = pm
  }
  
  return pm
}

func (es *EPServer) socketCheckIn(sktId string, padId string) {
  es.mu.Lock()
  defer es.mu.Unlock()
  es.skts[sktId] = padId
}

func (es *EPServer) socketCheckOut(sktId string) {
  es.mu.Lock()
  defer es.mu.Unlock()
  delete(es.skts, sktId)
}

func (es *EPServer) lookupPadId(sktId string) (string, bool) {
  es.mu.Lock()
  defer es.mu.Unlock()
  pid, ok := es.skts[sktId]
  return pid, ok
}

func (es *EPServer) processOp(padId string, op Op) {
  es.mu.Lock()
  defer es.mu.Unlock()

  newId := int64(0)
  for newId == 0 {
    newId = nrand()
  }
  le := PxLogEntry{newId, padId, op}
  log.Printf("1")
  es.paxosLogConsolidate()
  log.Printf("2")
  seq := es.paxosAppendToLog(le)
  log.Printf("3")
  es.applyLog(seq)
  log.Printf("4")
}

func NewEPServer(pxpeers []string, me int, sio *socketio.Server) *EPServer {
  gob.Register(PxLogEntry{})

  es := &EPServer{}
  es.sio = sio
  es.px = paxos.Make(pxpeers, me, nil)
  es.skts = make(map[string]string)
  es.pads = make(map[string]*PadManager)
  es.commitPoint = 0

  return es
}

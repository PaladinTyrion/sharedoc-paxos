package "main"

import (
  "sync"
  "log"
  "crypto/rand"
  "paxos"
  "github.com/googollee/go-socket.io"
)

type EPServer struct {
  mu          sync.Mutex
  sio         *socketio.Server
  px          *paxos.Paxos
  skts        map[string]int64     // socket id -> pad id
                                   // live session information
  pads        map[int64]PadManager // pad id -> the actual etherpad
                                   // manager, paxos-agreed state
  commitPoint int
}

func nrand() int64 {
  max := big.NewInt(int64(1) << 62)
  bigx, _ := rand.Int(rand.Reader, max)
  x := bigx.Int64()
  return x
}

func (es *EPServer) getPadById(padId int64) *PadManager {
  es.mu.Lock()
  defer es.mu.Unlock()

  pm, ok := es.pads[padId]
  if !ok {
    pm = NewPadManager(padId)
    es.pads[padId] = pm
  }
  
  return &pm
}

func (es *EPServer) socketCheckIn(sktId string, padId int64) {
  es.mu.Lock()
  defer es.mu.Unlock()
  es.skts[sktId] = padId
}

func (es *EPServer) socketCheckOut(sktId string) {
  es.mu.Lock()
  defer es.mu.Unlock()
  delete(es.skts, sktId)
}

func (es *EPServer) lookupPadId(sktId string) (int64, bool) {
  es.mu.Lock()
  defer es.mu.Unlock()
  pid, ok := es.skts[sktId]
  return pid, ok
}

func (es *EPServer) processOp(padId int64, op Op) {
  es.mu.Lock()
  defer es.mu.Unlock()

  le := PxLogEntry{nrand(), padId, op}
  es.paxosLogConsolidate()
  seq := es.paxosAppendToLog(le)
  es.applyLog(seq)
}

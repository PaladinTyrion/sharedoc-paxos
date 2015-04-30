package "main"

import (
  "sync"
  "log"
  "paxos"
  "github.com/googollee/go-socket.io"
)

type EPServer struct {
  mu   sync.Mutex
  sio  *socketio.Server
  // do we need more than one paxos?
  // one for client requests one for committed operations?
  px   *paxos.Paxos
  pads map[int64]PadManager // pad id -> the actual etherpad manager
}

func (es *EPServer) getPadById(padId int64) *PadManager {
  es.mu.Lock()
  defer es.mu.Unlock()

  pm, ok := es.pads[padId]
  if !ok {
    pm := NewPadManager(padId)
    es.pads[padId] = pm
  }

  return &pm
}
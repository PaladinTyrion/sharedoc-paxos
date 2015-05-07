package main

import (
  "encoding/json"
  "time"
  "paxos"
)

// Paxos utilities
// Require Mutex be held!

type SOp struct {
  ID       int64
  Version  uint64
  Type     string
  Position uint64
  Value    string
}

type PxLogEntry struct {
  EntryId  int64
  PadId    string
  ClientOp Op
}

var const_noop PxLogEntry = PxLogEntry{int64(0), "", Op{}}

// EPServer::startAndWait():
// Start a paxos agreement at instance number seq, and wait until
// consensus is reached. Returns the decided log entry at that seq.
// This function is only to be called by paxosAppendLog() and
// paxosLogConsolidate().
func (es *EPServer) startAndWait(seq int, le PxLogEntry) PxLogEntry {
  to := 10 * time.Millisecond
  es.px.Start(seq, le)
  for {
    status, v := es.px.Status(seq)
    if status == paxos.Decided {
      // Always return with mutex held
      return v.(PxLogEntry)
    }
    time.Sleep(to)
    // Exponential backoff
    if to < 1*time.Second {
      to *= 2
    }
  }
}

// EPServer::paxosAppendToLog():
// Appends LogEntry to the end of the paxos log known to this server,
// and returns the position in log of the appended entry
//
// It can be shown by induction that this eager appending technique
// leaves no holes in the log.
func (es *EPServer) paxosAppendToLog(le PxLogEntry) int {
  seq := es.px.Max() + 1
  for {
    var temp PxLogEntry
    status, v := es.px.Status(seq)
    if status != paxos.Decided {
      temp = es.startAndWait(seq, le)
    } else {
      temp = v.(PxLogEntry)
    }
    if temp.EntryId == le.EntryId {
      // succeeds!
      return seq
    }
    seq++
  }
}

// EPServer::paxosLogConsolidate():
// Fill up potential holes in unapplied log (after commitPoint). Just
// to be conservative.
func (es *EPServer) paxosLogConsolidate() {
  es.paxosLogConsolidate_explicit(es.px.Max())
}

func (es *EPServer) paxosLogConsolidate_explicit(upto int) {
  for i := es.commitPoint; i <= upto; i++ {
    status, _ := es.px.Status(i)
    if status == paxos.Decided {
      continue
    } else {
      // insert a noop (this should never be executed)
      es.startAndWait(i, const_noop)
    }
  }
}

// EPServer::applyLog():
// Apply all log entries with seq between the oldest (unapplied) entry
// and ceiling, indices are inclusive, assuming no holes between
// oldest entry and ceiling. Advances es.commitPoint.
func (es *EPServer) applyLog(ceiling int) {
  for es.commitPoint <= ceiling {
    status, le := es.px.Status(es.commitPoint)
    assert(status == paxos.Decided, "applyLog")

    es.applyEntry(le.(PxLogEntry))
    es.commitPoint++
  }
  es.px.Done(ceiling)
}

// EPServer::applyEntry():
// Update etherpad state and broadcast the committed operation to all
// sockets connected to this etherpad. Note that different clients
// could be connected to different paxos peers
func (es *EPServer) applyEntry(le PxLogEntry) {
  if le.EntryId == int64(0) {
    return
  }

  pm, ok := es.pads[le.PadId]
  if !ok {
    pm = NewPadManager(le.PadId)
    es.pads[le.PadId] = pm
  }
  cop := pm.registerOp(le.ClientOp)
  ncop := toStringOp(cop)
  opJSON, err := json.Marshal(ncop)
  assert(err == nil, "panic 2")
  //log.Printf("broadcast %v\n", string(opJSON[:]))
  es.sio.BroadcastTo(le.PadId, "op", string(opJSON[:]))
}

func toStringOp(opIn Op) SOp {
  ret := SOp{opIn.ID, opIn.Version, "", opIn.Position, opIn.Value}
  if opIn.Type == InsertOp {
    ret.Type = "Insert"
  } else {
    ret.Type = "Delete"
  }
  return ret
}

// call this in a separate Goroutine in a loop, with timer delays
func (es *EPServer) autoApply() {
  es.mu.Lock()
  defer es.mu.Unlock()

  max := es.px.MaxKnown()

  if max < es.commitPoint {
    return
  }

  es.paxosLogConsolidate_explicit(max)
  es.applyLog(max)
  return
}

func (es *EPServer) startAutoApply() {
  go func () {
    for {
      es.autoApply()
      time.Sleep(100 * time.Millisecond)
    }
  }()
}

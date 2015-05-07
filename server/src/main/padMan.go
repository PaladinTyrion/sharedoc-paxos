package main

import (
  "log"
  "sync"
)

type PadInfo struct {
  PadId   string
  Version uint64
  Ops     []SOp
}

type PadManager struct {
  // mutex is protecting against operations that don't go through Paxos
  mu      sync.Mutex
  padId   string
  rev     uint64
  text    string
  history map[uint64]Op // revision base -> committed Op
}

// PadManager::registerOp()
// Takes an incoming client operation, reconcile it with conflicting
// operations (if any), and emits the committed version of the same
// operation. Assumes that operations are passed in in paxos-log order
// without duplications.
func (pm *PadManager) registerOp(opIn Op) Op {
  pm.mu.Lock()
  defer pm.mu.Unlock()

  opRet := opIn
  if opIn.Version == pm.rev {
    // No conflicts, apply it direcly
    log.Printf("committed %v", opIn)
    pm.applyCommittedOp(opIn)
    log.Printf("server version %v", pm.rev)
  } else {
    log.Printf("server version %v", pm.rev)
    assert(opIn.Version < pm.rev, "RegisterOp")
    for v := opIn.Version; v < pm.rev; v++ {
      opReconcile(&opRet, pm.history[v])
    }
    pm.applyCommittedOp(opRet)
  }

  return opRet
}

// PadManager::applyCommittedOp()
// Applies a committed operation to update etherpad state.
func (pm *PadManager) applyCommittedOp(op Op) {
  assert(op.Version == pm.rev, "applyCommittedOp")
  /*
  if op.Type == InsertOp {
    if op.Position < 0 {
      pm.text = op.Value[0:0] + pm.text
    } else if op.Position >= uint64(len(pm.text)) {
      pm.text += op.Value[0:0]
    } else {
      pm.text = pm.text[:op.Position] + op.Value[0:0] + pm.text[op.Position+1:]
    }
  } else if op.Type == DeleteOp {
    if op.Position == 0 {
      pm.text = pm.text[1:]
    } else if op.Position == uint64(len(pm.text)) {
      pm.text = pm.text[:len(pm.text)-1]
    } else {
      pm.text = pm.text[:op.Position-1] + pm.text[op.Position+1:]
    }
  } else {
    // noop, do nothing
  }
  */

  _, ok := pm.history[pm.rev]
  assert(!ok, "applyCommittedOp - rev exists")
  pm.history[pm.rev] = op
  pm.rev++

  return
}

// opReconcile()
// Reconcile (uncommitted) op1 with committed operation op2, retaining
// the intentions of both operations. op2 is never changed (because
// it's committed), and op1 will be updated in-place. Note that the
// updated op1 is still uncommitted. It's necessary to run this
// function in a loop to iteratively merge op1 with all committed
// operations to transform op1 into a committed operation.
func opReconcile(op1 *Op, op2 Op) {
  assert(op1.Version == op2.Version, "opReconcile - rev")
  if op1.Type == InsertOp {
    if op2.Type == InsertOp {
      // insert vs. insert
      if op2.Position <= op1.Position {
        op1.Position++
      } else {
        // nothing to be done here
      }
    } else if op2.Type == DeleteOp {
      // insert vs. delete
      if op2.Position <= op1.Position {
        op1.Position--
      } else {
        // do nothing
      }
    } else {
      // op2 is noop, everything fine
    }
  } else if op1.Type == DeleteOp {
    if op2.Type == InsertOp {
      // delete vs. insert
      if op2.Position < op1.Position {
        op1.Position++
      } else {
        // nothing to be done here
      }
    } else if op2.Type == DeleteOp {
      // delete vs. delete, be extra careful here
      if op2.Position < op1.Position {
        op1.Position--
      } else if op2.Position == op1.Position {
        op1.Type = NoOp
      } else {
        // do nothing
      }
    } else {
      // op2 is noop, everything fine
    }
  } else {
    // once a noop, always a noop; do nothing
  }
  op1.Version++
  return
}

func (pm *PadManager) getLatestInfo() PadInfo {
  pm.mu.Lock()
  defer pm.mu.Unlock()
  
  ret := PadInfo{}
  ret.PadId = pm.padId
  ret.Version = pm.rev
  ret.Ops = make([]SOp, 0)
  for i := uint64(0); i < pm.rev; i++ {
    ret.Ops = append(ret.Ops, toStringOp(pm.history[i]))
  }

  return ret
}

func NewPadManager(padId string) *PadManager {
  pm := PadManager{}

  pm.padId = padId
  pm.rev = uint64(0)
  pm.text = ""
  pm.history = make(map[uint64]Op)

  return &pm
}

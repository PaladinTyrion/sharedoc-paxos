package "main"

import (
  "sync"
)

type PadInfo struct {
  PadId int64
  Rev   uint64
  Text  string
}

type PadManager struct {
  // mutex is protecting against operations that don't go through Paxos
  mu      sync.Mutex
  padId   int64
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
  if opIn.Rev == pm.rev {
    // No conflicts, apply it direcly
    pm.applyCommittedOp(opIn)
  } else {
    assert(opIn.Rev < pm.rev, "RegisterOp")
    for v := opIn.Rev; v < pm.rev; v++ {
      opReconcile(&opRet, pm.history[v])
    }
    pm.applyCommittedOp(opRet)
  }

  return opRet
}

// PadManager::applyCommittedOp()
// Applies a committed operation to update etherpad state.
func (pm *PadManager) applyCommittedOp(op Op) {
  assert(opIn.Rev == pm.rev, "applyCommittedOp")
  
  if op.Opty == InsertOp {
    if op.Pos < 0 {
      pm.text = op.Char[0] + pm.text
    } else if op.Pos >= len(pm.text) {
      pm.text += op.Char[0]
    } else {
      pm.text = pm.text[:op.Pos] + op.Char[0] + pm.text[op.Pos+1:]
    }
  } else if op.Opty == DeleteOp {
    if op.Pos == 0 {
      pm.text = pm.text[1:]
    } else if op.Pos == len(pm.text) {
      pm.text = pm.text[:len(pm.text)-1]
    } else {
      pm.text = pm.text[:op.Pos-1] + pm.text[op.Pos+1:]
    }
  } else {
    // noop, do nothing
  }

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
  assert(op1.Rev == op2.Rev, "opReconcile - rev")
  if op1.Opty == InsertOp {
    if op2.Opty == InsertOp {
      // insert vs. insert
      if op2.Pos <= op1.Pos {
        op1.Pos++
      } else {
        // nothing to be done here
      }
    } else if op2.Opty == DeleteOp {
      // insert vs. delete
      if op2.Pos <= op1.Pos {
        op1.Pos--
      } else {
        // do nothing
      }
    } else {
      // op2 is noop, everything fine
    }
  } else if op1.Opty == DeleteOp {
    if op2.Opty == InsertOp {
      // delete vs. insert
      if op2.Pos < op1.Pos {
        op1.Pos++
      } else {
        // nothing to be done here
      }
    } else if op2.Opty == DeleteOp {
      // delete vs. delete, be extra careful here
      if op2.Pos < op1.Pos {
        op1.Pos--
      } else if op2.Pos == op1.Pos {
        op1.Opty = NoOp
      } else {
        // do nothing
      }
    } else {
      // op2 is noop, everything fine
    }
  } else {
    // once a noop, always a noop; do nothing
  }
  op1.Rev++
  return
}

func (pm *PadManager) getLatestInfo() PadInfo {
  pm.mu.Lock()
  defer pm.mu.Unlock()
  
  ret := PadInfo{}
  ret.PadId = pm.padId
  ret.Rev = pm.rev
  ret.Text = pm.text

  return ret
}

func NewPadManager(padId int64) PadManager {
  pm := PadManager{}

  pm.padId = padId
  pm.rev = uint64(0)
  pm.text = ""
  pm.history = make(map[uint64]Op)

  return pm
}

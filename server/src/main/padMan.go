package "main"

type PadManager struct {
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
  opRet := opIn
  if opIn.Rev == pm.rev {
    // No conflicts, apply it direcly
    pm.applyCommittedOp(opIn)
  } else {
    assert(opIn.Rev < pm.rev, "RegisterOp")
    for v := opIn.Rev; v < pm.rev; v++ {
      opReconcile(&opRet, pm.history[v])
      if opRet.Opty == NoOp {
        break
      }
    }
    pm.applyCommittedOp(opRet)
  }

  return opRet
}

// PadManager::applyCommittedOp()
// Applies a committed operation to update server state.
func (pm *PadManager) applyCommittedOp(op Op) {
  if op.Opty == NoOp {
    return
  }

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
    assert(false, "applyCommittedOp - Opty")
  }

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
      // panic
      assert(false, "opReconcile - committed noop1")
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
      assert(false, "opReconcile - committed noop2")
    }
  } else {
    assert(false, "opReconcile - op1 noop")
  }
  op1.Rev++
  return
}

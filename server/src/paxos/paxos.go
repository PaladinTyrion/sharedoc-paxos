package paxos

//
// Paxos library, to be included in an application.
// Multiple applications will run, each including
// a Paxos peer.
//
// Manages a sequence of agreed-on values.
// The set of peers is fixed.
// Copes with network failures (partition, msg loss, &c).
// Does not store anything persistently, so cannot handle crash+restart.
//
// The application interface:
//
// px = paxos.Make(peers []string, me string)
// px.Start(seq int, v interface{}) -- start agreement on new instance
// px.Status(seq int) (Fate, v interface{}) -- get info about an instance
// px.Done(seq int) -- ok to forget all instances <= seq
// px.Max() int -- highest instance seq known, or -1
// px.Min() int -- instances before this seq have been forgotten
//

import (
  "net"
  "net/rpc"
  "log"
  "os"
  "syscall"
  "sync"
  "sync/atomic"
  "fmt"
  "math/rand"
  "time"
)

// px.Status() return values, indicating
// whether an agreement has been decided,
// or Paxos has not yet reached agreement,
// or it was agreed but forgotten (i.e. < Min()).

const (
  Decided   int32 = iota + 1
  Pending        // not yet decided.
  Forgotten      // decided but forgotten.
)

const (
  OK       = 0
  Rejected = 1
)

// The two-part proposal number structure
type ProposalNumber struct {
  PN int // Per-machine proposal number
  ID int // Machine ID
}

// This defines the total order among proposal numbers
func (lhs *ProposalNumber) higherThan(rhs *ProposalNumber) bool {
  return lhs.PN > rhs.PN || (lhs.PN == rhs.PN && lhs.ID > rhs.ID)
}

func (lhs *ProposalNumber) geq(rhs *ProposalNumber) bool {
  return lhs.higherThan(rhs) || *lhs == *rhs
}

func (pn *ProposalNumber) isNil() bool {
  return pn.PN == 0 && pn.ID == 0
}

// Paxos instance structs. DO NOT use in RPCs!
type PxAcceptorInstance struct {
  mu         sync.Mutex
  maxPrepare ProposalNumber
  maxAccept  ProposalNumber
  value      interface{}
}

type PxProposerInstance struct {
  mu     sync.Mutex
  status int32        // atmoic access
  value  interface{}
}

// RPC argument and reply formats
type PrepareArgs struct {
  Seq int
  N   ProposalNumber
}

type PrepareReply struct {
  Result   int
  PNHint   ProposalNumber
  AccMax   ProposalNumber
  Value    interface{}
  DoneUpTo int
}

type AcceptArgs struct {
  Seq int
  N   ProposalNumber
  V   interface{}
}

type AcceptReply struct {
  Result   int
  DoneUpTo int
}

type DecidedArgs struct {
  Seq int
  V   interface{}
}

type DecidedReply struct {
  DoneUpTo int
}

type MinimumSet struct {
  mu   sync.Mutex
  vals []int
}

func minimumSetInit(size int) MinimumSet {
  set := &MinimumSet{}
  set.vals = make([]int, size)
  for i := 0; i < size; i++ {
    set.vals[i] = -1
  }
  return *set
}

func (ms *MinimumSet) setVal(at int, val int) bool {
  ms.mu.Lock()
  if at < 0 || at >= len(ms.vals) {
    log.Printf("Array access out of bound!\n")
    ms.mu.Unlock()
    return false
  }
  if val < ms.vals[at] {
    // log.Printf("Out of date done value!\n")
    ms.mu.Unlock()
    return true
  }
  ms.vals[at] = val
  ms.mu.Unlock()
  return true
}

func (ms *MinimumSet) getVal(at int) int {
  ms.mu.Lock()
  if at < 0 || at >= len(ms.vals) {
    log.Printf("Array access out of bound!\n")
    return 0
  }
  v := ms.vals[at]
  ms.mu.Unlock()
  return v
}

func (ms *MinimumSet) getMin() int {
  ms.mu.Lock()
  r := ms.vals[0]
  for _, v := range ms.vals {
    if v < r {
      r = v
    }
  }
  ms.mu.Unlock()
  return r
}

type Paxos struct {
  mu         sync.Mutex
  l          net.Listener
  dead       int32 // for testing
  unreliable int32 // for testing
  rpcCount   int32 // for testing
  peers      []string
  me         int // index into peers[]

  // Your data here.
  // Acceptor and proposer's states should be kept separate
  acceptorInstances map[int]*PxAcceptorInstance
  proposerInstances map[int]*PxProposerInstance
  peerDones         MinimumSet
  maxSeq            int // The max sequence number proposed by this peer
  maxKnownSeq       int // The max sequence number in the maps above
  minSeq            int // The min sequence number in the maps above
}

func newAcceptorInstance() *PxAcceptorInstance {
  ins := &PxAcceptorInstance{}
  ins.maxPrepare = ProposalNumber{0, 0}
  ins.maxAccept = ins.maxPrepare
  return ins
}

func newProposerInstance() *PxProposerInstance {
  ins := &PxProposerInstance{}
  atomic.StoreInt32(&ins.status, Pending)
  return ins
}

func (px *Paxos) findAcceptorInstance(seq int) *PxAcceptorInstance {
  ins, ok := px.acceptorInstances[seq]
  if !ok {
    return nil
  }
  return ins
}

func (px *Paxos) findProposerInstance(seq int) *PxProposerInstance {
  ins, ok := px.proposerInstances[seq]
  if !ok {
    return nil
  }
  return ins
}

func (px *Paxos) getAcceptorInstance(seq int) *PxAcceptorInstance {
  px.mu.Lock()
  ins := px.findAcceptorInstance(seq)
  if ins == nil {
    ins = newAcceptorInstance()
    px.acceptorInstances[seq] = ins
  }
  px.mu.Unlock()
  return ins
}

func (px *Paxos) getProposerInstance(seq int) *PxProposerInstance {
  px.mu.Lock() 
  ins := px.findProposerInstance(seq)
  if ins == nil {
    ins = newProposerInstance()
    px.proposerInstances[seq] = ins
  }
  px.mu.Unlock()
  return ins
}

// RPC Handlers
func (px *Paxos) Prepare(args *PrepareArgs, reply *PrepareReply) error {
  ins := px.getAcceptorInstance(args.Seq)
  ins.mu.Lock()

  reply.AccMax = ins.maxAccept
  reply.Value = ins.value
  reply.DoneUpTo = px.peerDones.getVal(px.me)
  if args.N.higherThan(&ins.maxPrepare) {
    ins.maxPrepare = args.N
    reply.Result = OK
    reply.PNHint = ins.maxPrepare
  } else {
    reply.Result = Rejected
    reply.PNHint = ins.maxPrepare
  }

  ins.mu.Unlock()
  return nil
}

func (px *Paxos) Accept(args *AcceptArgs, reply *AcceptReply) error {
  ins := px.getAcceptorInstance(args.Seq)
  ins.mu.Lock()

  reply.DoneUpTo = px.peerDones.getVal(px.me)
  if args.N.geq(&ins.maxPrepare) {
    ins.maxPrepare = args.N
    ins.maxAccept = args.N
    ins.value = args.V
    reply.Result = OK
  } else {
    reply.Result = Rejected
  }

  ins.mu.Unlock()
  return nil
}

func (px *Paxos) Decided(args *DecidedArgs, reply *DecidedReply) error {
  ins := px.getProposerInstance(args.Seq)
  if atomic.LoadInt32(&ins.status) != Decided {
    
    px.mu.Lock()
    if args.Seq > px.maxKnownSeq {
      px.maxKnownSeq = args.Seq
    }
    px.mu.Unlock()
    
    ins.value = args.V
    atomic.StoreInt32(&ins.status, Decided)
  }
  reply.DoneUpTo = px.peerDones.getVal(px.me)
  return nil
}

//
// call() sends an RPC to the rpcname handler on server srv
// with arguments args, waits for the reply, and leaves the
// reply in reply. the reply argument should be a pointer
// to a reply structure.
//
// the return value is true if the server responded, and false
// if call() was not able to contact the server. in particular,
// the replys contents are only valid if call() returned true.
//
// you should assume that call() will time out and return an
// error after a while if it does not get a reply from the server.
//
// please use call() to send all RPCs, in client.go and server.go.
// please do not change this function.
//
func call(srv string, name string, args interface{}, reply interface{}) bool {
  c, err := rpc.Dial("unix", srv)
  if err != nil {
    err1 := err.(*net.OpError)
    if err1.Err != syscall.ENOENT && err1.Err != syscall.ECONNREFUSED {
      fmt.Printf("paxos Dial() failed: %v\n", err1)
    }
    return false
  }
  defer c.Close()

  err = c.Call(name, args, reply)
  if err == nil {
    return true
  }

  fmt.Println(err)
  return false
}

func assert(cond bool) {
  if !cond {
    log.Printf("Assertion failed! Aborting.\n")
    os.Exit(1)
  }
  return
}

type ConnectorLocalStats struct {
  mu        sync.Mutex
  okCount   int
  maxPSoFar ProposalNumber
  maxASoFar ProposalNumber
  maxV      interface{}
}

func (s *ConnectorLocalStats) registerOK(acc ProposalNumber, val interface{}) {
  s.mu.Lock()
  s.okCount++
  if acc.higherThan(&s.maxASoFar) {
    s.maxASoFar = acc
    s.maxV = val
  }
  s.mu.Unlock()
}

func (s *ConnectorLocalStats) registerRej(pre ProposalNumber) {
  s.mu.Lock()
  if pre.higherThan(&s.maxPSoFar) {
    s.maxPSoFar = pre
  }
  s.mu.Unlock()
}

// The second return value represents the highest n_p in case of consensus failure
// and represents the highest n_a when consensus is reached
func (px *Paxos) sendPrepares(seq int, n ProposalNumber) (bool, ProposalNumber, interface{}) {
  args := &PrepareArgs{seq, n}
  results := ConnectorLocalStats{}
  results.okCount = 0
  results.maxPSoFar = ProposalNumber{0, 0}
  results.maxASoFar = results.maxPSoFar
  majority := len(px.peers) / 2

  var wg sync.WaitGroup
  wg.Add(len(px.peers))

  for idx, peer := range px.peers {
    if idx == px.me {
      reply := PrepareReply{}
      px.Prepare(args, &reply)
      if reply.Result == OK {
        results.registerOK(reply.AccMax, reply.Value)
      } else {
        results.registerRej(reply.PNHint)
      }
      wg.Done()
    } else {
      go func(peer string, idx int) {
        reply := PrepareReply{}
        ok := call(peer, "Paxos.Prepare", args, &reply)
        if ok {
          if reply.Result == OK {
            results.registerOK(reply.AccMax, reply.Value)
          } else {
            results.registerRej(reply.PNHint)
          }
          assert(px.peerDones.setVal(idx, reply.DoneUpTo))
        }
        wg.Done()
        // No need to retry in case of communication failure
        // Paxos does the math for us!
      }(peer, idx)
    }
  }

  wg.Wait()

  var maxRet ProposalNumber
  if results.okCount > majority {
    maxRet = results.maxASoFar
  } else {
    maxRet = results.maxPSoFar
  }

  return results.okCount > majority, maxRet, results.maxV
}

type ThreadSafeInt struct {
  mu    sync.Mutex
  value int
}

func (tsi *ThreadSafeInt) inc() {
  tsi.mu.Lock()
  tsi.value++
  tsi.mu.Unlock()
}

func (px *Paxos) sendAccepts(seq int, n ProposalNumber, v interface{}) bool {
  args := &AcceptArgs{seq, n, v}
  okCount := ThreadSafeInt{}
  okCount.value = 0
  majority := len(px.peers) / 2

  var wg sync.WaitGroup
  wg.Add(len(px.peers))

  for idx, peer := range px.peers {
    if idx == px.me {
      reply := AcceptReply{}
      px.Accept(args, &reply)
      if reply.Result == OK {
        okCount.inc()
      }
      wg.Done()
    } else {
      go func(peer string, idx int) {
        reply := AcceptReply{}
        ok := call(peer, "Paxos.Accept", args, &reply)
        if ok {
          assert(px.peerDones.setVal(idx, reply.DoneUpTo))
          if reply.Result == OK {
            okCount.inc()
          }
        }
        wg.Done()
      }(peer, idx)
    }
  }

  wg.Wait()

  return okCount.value > majority
}

// SendDecideds does not send Decided messages to itself to prevent deadlock
func (px *Paxos) sendDecideds(seq int, v interface{}) {
  args := &DecidedArgs{seq, v}
  for idx, peer := range px.peers {
    if idx != px.me {
      go func(peer string, idx int) {
        reply := DecidedReply{}
        ok := call(peer, "Paxos.Decided", args, &reply)
        if ok {
          assert(px.peerDones.setVal(idx, reply.DoneUpTo))
        }
      }(peer, idx)
    }
  }
}

func (px *Paxos) runProposer(seq int, ins *PxProposerInstance, v interface{}) {
  // Locking needed to make sure only one proposer is running at a time
  // for a given Paxos instance
  ins.mu.Lock()

  n := ProposalNumber{0, px.me}
  for atomic.LoadInt32(&ins.status) == Pending {
    n.PN++
    ok, maxRet, maxV := px.sendPrepares(seq, n)
    if ok {
      vPropose := v
      if !maxRet.isNil() {
        vPropose = maxV
      }

      if px.sendAccepts(seq, n, vPropose) {
        ins.value = vPropose
        atomic.StoreInt32(&ins.status, Decided)
        px.sendDecideds(seq, vPropose)
      }
    } else {
      // When rejected, the returned maxRet is of a hint for choosing n
      // It contains the highest prepare seen (n_p) as returned
      // among all servers that responded
      n.PN = maxRet.PN
    }

    if px.isdead() {
      // Die when asked to
      break
    }
  }

  ins.mu.Unlock()
  return
}

//
// the application wants paxos to start agreement on
// instance seq, with proposed value v.
// Start() returns right away; the application will
// call Status() to find out if/when agreement
// is reached.
//
func (px *Paxos) Start(seq int, v interface{}) {
  if seq < px.Min() {
    return
  }

  px.mu.Lock()
  if seq > px.maxSeq {
    px.maxSeq = seq
  }
  if px.maxSeq > px.maxKnownSeq {
    px.maxKnownSeq = px.maxSeq
  }
  px.mu.Unlock()

  ins := px.getProposerInstance(seq)
  go px.runProposer(seq, ins, v)
  return
}

//
// the application on this machine is done with
// all instances <= seq.
//
// see the comments for Min() for more explanation.
//
func (px *Paxos) Done(seq int) {
  if seq < px.peerDones.getVal(px.me) {
    return
  }
  assert(px.peerDones.setVal(px.me, seq))
}

func (px *Paxos) garbageCollect() {
  workingMin := px.peerDones.getMin()
  px.mu.Lock()
  for i := px.minSeq; i <= workingMin; i++ {
    delete(px.acceptorInstances, i)
    delete(px.proposerInstances, i)
  }
  px.minSeq = workingMin + 1
  px.mu.Unlock()
}

//
// the application wants to know the
// highest instance sequence known to
// this peer.
//
func (px *Paxos) Max() int {
  px.mu.Lock()
  defer px.mu.Unlock()
  return px.maxSeq
}

// returns the maximum known instance number
// including decided instances
func (px *Paxos) MaxKnown() int {
	px.mu.Lock()
	defer px.mu.Unlock()
	return px.maxKnownSeq
}

//
// Min() should return one more than the minimum among z_i,
// where z_i is the highest number ever passed
// to Done() on peer i. A peers z_i is -1 if it has
// never called Done().
//
// Paxos is required to have forgotten all information
// about any instances it knows that are < Min().
// The point is to free up memory in long-running
// Paxos-based servers.
//
// Paxos peers need to exchange their highest Done()
// arguments in order to implement Min(). These
// exchanges can be piggybacked on ordinary Paxos
// agreement protocol messages, so it is OK if one
// peers Min does not reflect another Peers Done()
// until after the next instance is agreed to.
//
// The fact that Min() is defined as a minimum over
// *all* Paxos peers means that Min() cannot increase until
// all peers have been heard from. So if a peer is dead
// or unreachable, other peers Min()s will not increase
// even if all reachable peers call Done. The reason for
// this is that when the unreachable peer comes back to
// life, it will need to catch up on instances that it
// missed -- the other peers therefor cannot forget these
// instances.
//
func (px *Paxos) Min() int {
  return px.peerDones.getMin() + 1
}

//
// the application wants to know whether this
// peer thinks an instance has been decided,
// and if so what the agreed value is. Status()
// should just inspect the local peer state;
// it should not contact other Paxos peers.
//
func (px *Paxos) Status(seq int) (int32, interface{}) {
  if seq < px.Min() {
    return Forgotten, nil
  }
  ins := px.findProposerInstance(seq)
  if ins == nil {
    return Forgotten, nil
  }
  var v interface{}
  st := atomic.LoadInt32(&ins.status)
  if st == Decided {
    v = ins.value
  } else {
    v = nil
  }
  return st, v
}

//
// tell the peer to shut itself down.
// for testing.
// please do not change these two functions.
//
func (px *Paxos) Kill() {
  atomic.StoreInt32(&px.dead, 1)
  if px.l != nil {
    px.l.Close()
  }
}

//
// has this peer been asked to shut down?
//
func (px *Paxos) isdead() bool {
  return atomic.LoadInt32(&px.dead) != 0
}

// please do not change these two functions.
func (px *Paxos) setunreliable(what bool) {
  if what {
    atomic.StoreInt32(&px.unreliable, 1)
  } else {
    atomic.StoreInt32(&px.unreliable, 0)
  }
}

func (px *Paxos) isunreliable() bool {
  return atomic.LoadInt32(&px.unreliable) != 0
}

//
// the application wants to create a paxos peer.
// the ports of all the paxos peers (including this one)
// are in peers[]. this servers port is peers[me].
//
func Make(peers []string, me int, rpcs *rpc.Server) *Paxos {
  px := &Paxos{}
  px.peers = peers
  px.me = me

  // Your initialization code here.
  px.acceptorInstances = make(map[int]*PxAcceptorInstance)
  px.proposerInstances = make(map[int]*PxProposerInstance)
  px.peerDones = minimumSetInit(len(px.peers))
  px.maxSeq = -1
  px.maxKnownSeq = -1
  px.minSeq = 0

  // Start garbage collection thread
  go func() {
    for px.isdead() == false {
      px.garbageCollect()
      time.Sleep(100 * time.Millisecond)
    }
  }()

  if rpcs != nil {
    // caller will create socket &c
    rpcs.Register(px)
  } else {
    rpcs = rpc.NewServer()
    rpcs.Register(px)

    // prepare to receive connections from clients.
    // change "unix" to "tcp" to use over a network.
    os.Remove(peers[me]) // only needed for "unix"
    l, e := net.Listen("unix", peers[me])
    if e != nil {
      log.Fatal("listen error: ", e)
    }
    px.l = l

    // please do not change any of the following code,
    // or do anything to subvert it.

    // create a thread to accept RPC connections
    go func() {
      for px.isdead() == false {
        conn, err := px.l.Accept()
        if err == nil && px.isdead() == false {
          if px.isunreliable() && (rand.Int63()%1000) < 100 {
            // discard the request.
            conn.Close()
          } else if px.isunreliable() && (rand.Int63()%1000) < 200 {
            // process the request but force discard of reply.
            c1 := conn.(*net.UnixConn)
            f, _ := c1.File()
            err := syscall.Shutdown(int(f.Fd()), syscall.SHUT_WR)
            if err != nil {
              fmt.Printf("shutdown: %v\n", err)
            }
            atomic.AddInt32(&px.rpcCount, 1)
            go rpcs.ServeConn(conn)
          } else {
            atomic.AddInt32(&px.rpcCount, 1)
            go rpcs.ServeConn(conn)
          }
        } else if err == nil {
          conn.Close()
        }
        if err != nil && px.isdead() == false {
          fmt.Printf("Paxos(%v) accept: %v\n", me, err.Error())
        }
      }
    }()
  }

  return px
}

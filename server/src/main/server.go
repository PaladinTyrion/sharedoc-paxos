package main

import (
  "os"
  "log"
  "fmt"
  "sync"
  "strconv"
  "errors"
  "encoding/json"
  "net/http"
  "github.com/googollee/go-socket.io"
)

const (
  PXCONFIG = 3
)

// boring parsing stuff 1.0
func spawnServer(pxpeers []string, me int, wg *sync.WaitGroup) {
  server, err := socketio.NewServer(nil)
  if err != nil {
      log.Fatal(err)
  }

  // we need the server argument because paxos needs it to send
  // broadcast messages when an operation is committed
  es := NewEPServer(pxpeers, me, server)
  
  server.On("connection", func(so socketio.Socket) {
    // Client should first send a "open pad" message, with "pad id"
    // (an integer in string format) as the argument
    // all subsequent edits are assumed to be operating on this pad
    log.Printf("connection")
    so.On("open pad", func(pad string) {
      log.Printf("open pad %v\n", pad)
      if len(so.Rooms()) > 1 {
        so.Emit("error", "alreay opened")
        return
      }

      // wrapping mutex around it because socketio not thread-safe
      // this is cumbersome and should be fixed later
      es.mu.Lock()
      so.Join(pad)
      es.mu.Unlock()
      
      pm := es.getPadById(pad)
      es.socketCheckIn(so.Id(), pad)
      piJSON, err := json.Marshal(pm.getLatestInfo())
      assert(err == nil, "panic 1")
      so.Emit("init_comt_op", string(piJSON[:]))
      return
    })

    // An "op" message's argument is a JSON string with all string
    // fields. Field names should be kept consistent with Op{} in
    // common.go, case-sensitive.
    so.On("op", func(opJSON string) {
      log.Printf("received op %v\n", opJSON)
      padId, ok := es.lookupPadId(so.Id())
      if !ok {
        so.Emit("error", "not checked in")
        return
      }
      sOp := make(map[string]string)
      err := json.Unmarshal([]byte(opJSON), &sOp)
      if err != nil {
        so.Emit("error", "invalid op")
        return
      }
      op, err := toNativeOp(sOp)
      if err != nil {
        so.Emit("error", "invalid op")
        return
      }
      // do not emit anything here, use paxos to do the correct
      // thing when a committed operation is discovered
      es.processOp(padId, op)
      return
    })

    so.On("disconnection", func(){
      es.socketCheckOut(so.Id())
    })
  })
  
  server.On("error", func(so socketio.Socket, err error) {
      log.Println("error:", err)
  })

  es.startAutoApply()

  srvMux := http.NewServeMux()
  srvMux.Handle("/socket.io/", server)
  srvMux.Handle("/", http.FileServer(http.Dir("../../../socket_editting/public/")))
  port := 8080
  portStr := fmt.Sprintf(":%v", port+me)
  log.Printf("Server %v running at localhost%v\n", me, portStr)
  log.Fatal(http.ListenAndServe(portStr, srvMux))
  wg.Done()
}

// boring parsing stuff 2.0
func toNativeOp(sOp map[string]string) (Op, error) {
  var ret Op
  var v interface{}
  var err error

  v, err = checkAndParse("int64", "ID", sOp)
  if err != nil {
    return ret, err
  }
  ret.ID = v.(int64)
  
  v, err = checkAndParse("uint64", "Version", sOp)
  if err != nil {
    return ret, err
  }
  ret.Version = v.(uint64)
  
  if s, ok := sOp["Type"]; ok {
    var opcode int
    if s == "Insert" {
      opcode = InsertOp
    } else if s == "Delete" {
      opcode = DeleteOp
    } else {
      opcode = NoOp
    }
    ret.Type = opcode
  } else {
    return ret, errors.New("Key not found: Type")
  }
  
  v, err = checkAndParse("uint64", "Position", sOp)
  if err != nil {
    return ret, err
  }
  ret.Position = v.(uint64)
  
  v, err = checkAndParse("string", "Value", sOp)
  if err != nil {
    return ret, err
  }
  ret.Value = v.(string)

  return ret, err
}

func checkAndParse(dtype string, key string,
                   sOp map[string]string) (interface{}, error) {
  s, ok := sOp[key]
  if !ok {
    return nil, errors.New("Key not found: "+key)
  }
  
  var v interface{}
  var err error
  if dtype == "int64" {
    v, err = strconv.ParseInt(s, 10, 64)
  } else if dtype == "uint64" {
    v, err = strconv.ParseUint(s, 10, 64)
  } else if dtype == "string" {
    v, err = s, nil
  }
  if err != nil {
    return nil, errors.New(fmt.Sprintf("Invalid %v format", key))
  } else {
    return v, err
  }
}

func port(host int) string {
  s := fmt.Sprintf("/var/tmp/824proj-%v/", os.Getuid())
  os.Mkdir(s, 0777)
  s += fmt.Sprintf("px-%v-%v", os.Getpid(), host)
  return s
}

func main() {
  pxpeers := make([]string, 0)
  for i := 0; i < PXCONFIG; i++ {
    pxpeers = append(pxpeers, port(i))
  }

  var wg sync.WaitGroup
  for i := 0; i < PXCONFIG; i++ {
    wg.Add(1)
    go spawnServer(pxpeers, i, &wg)
  }
  wg.Wait()

  return
}

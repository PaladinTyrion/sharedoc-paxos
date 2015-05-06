package "main"

import (
  "log"
  "fmt"
  "strconv"
  "errors"
  "encoding/json"
  "net/http"
  "github.com/googollee/go-socket.io"
)

// boring parsing stuff 1.0
func spawnServer(pxpeers []string, me int) {
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
    so.On("open pad", func(pad string) {
      if len(so.Rooms) > 1 {
        so.Emit("error", "alreay opened")
        return
      }

      padId, err := strconv.ParseInt(pad, 10, 64)
      if err != nil {
        so.Emit("error", "invalid pad id")
      } else {
        so.Join(pad)
        pm := es.getPadById(padId)
        es.socketCheckIn(so.Id(), padId)
        piJSON, err := json.Marshal(pm.getLatestInfo())
        assert(err == nil, "panic 1")
        so.Emit("pad info", string(piJSON[:]))
      }
      return
    })

    // An "edit" message's argument is a JSON string with all string
    // fields. Field names should be kept consistent with Op{} in
    // common.go, case-sensitive.
    so.On("edit", func(opJSON string) {
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
      if op, err := toNativeOp(sOp); err != nil {
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

  http.Handle("/api/", server)
  port := 5000
  log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port+pxid), nil))
}

// boring parsing stuff 2.0
func toNativeOp(sOp map[string]string) (Op, error) {
  var ret Op
  var v interface{}
  var err error

  v, err = checkAndParse("int", "Cid", sOp)
  if err != nil {
    return ret, err
  }
  ret.Cid = v.(int)

  v, err = checkAndParse("int", "Seq", sOp)
  if err != nil {
    return ret, err
  }
  ret.Seq = v.(int)
  
  v, err = checkAndParse("uint64", "Rev", sOp)
  if err != nil {
    return ret, err
  }
  ret.Rev = v.(uint64)
  
  v, err = checkAndParse("int", "Opty", sOp)
  if err != nil {
    return ret, err
  }
  ret.Opty = v.(int)
  
  v, err = checkAndParse("uint64", "Pos", sOp)
  if err != nil {
    return ret, err
  }
  ret.Pos = v.(uint64)
  
  v, err = checkAndParse("string", "Char", sOp)
  if err != nil {
    return ret, err
  }
  ret.Char = v.(string)

  return ret, err
}

func checkAndParse(dtype string, key string,
                   sOp map[string]string) (interface{}, error) {
  s, ok := sOp[key]
  if !ok {
    return ret, errors.New("Key not found: "+key)
  }
  
  var v interface{}
  var err error
  if dtype == "int" {
    v, err = strconv.ParseInt(s, 10, 0)
  } else if dtype == "uint64" {
    v, err = strconv.ParseUInt(s, 10, 64)
  } else if dtype == "string" {
    v, err = s, nil
  }
  if err != nil {
    return ret, errors.New(fmt.Sprintf("Invalid %v format", key))
  } else {
    return v, err
  }
}

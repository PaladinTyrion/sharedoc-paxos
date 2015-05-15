# ShareDoc-Paxos
Etherpad-like online docs collaboration app with Paxos servers

**IMPORTANT MESSAGE TO ETHERPAD USERS: This project is NOT in any way related to [Etherpad](http://etherpad.org).**

This is a work-in-progress developed to be the final project of [6.824 (Spring 2015)](http://nil.csail.mit.edu/6.824/2015/) at Massachusetts Institute of Technology.

This project depends on [go-socket.io](https://github.com/googollee/go-socket.io) by googollee. We wish we can express our sincere thanks to all contributors in the open-source community whose hardwork made this project possible.

## Build Instructions
1. After cloning this repository, please make sure to retrieve all submodules:

   ```shell
   $ git submodule init && git submodule update
   ```

2. The server code is located in `server/src/main` and the client-side implementation can be found in `socket_editting/public`. Note that the client-side program in forms of HTML and JavaScript code is served to a client browser as static files. The server program looks for these files upon initialization so please do not move them.

3. To build the server, please first specify the correct `$GOPATH` environment variable:

   ```shell
   $ export GOPATH=$REPO_ROOT/server
   ```
   Please make sure to export the **absolute** path. For example, use `/home/ubuntu/git/etherpad-paxos/server` instead of `etherpad-paxos/server`.
   Go to `server/src/main` and invoke the Go compiler using the `go` command:

   ```shell
   $ cd server/src/main
   $ go build
  ```
  
  As in other web applications, no build is required for the client-side program.

4. The above commands will produce a executable called `main` in the working directory. Please make sure to only run the server in the directory where the server is compiled (`server/src/main`). After the server is up and running, it should be listening on the following three addresses (we use a three-server Paxos configuration by default):

   ```
   http://localhost:8080 # paxos peer 0
   http://localhost:8081 # paxos peer 1
   http://localhost:8082 # paxos peer 2
   ```

5. Direct your browser (tested on latest Chrome and Safari releases as of May 8, 2015) to any server address above and see it in action!

6. If you want to know how our design works without digging through the source code, see our project write-up in the `docs` directory.

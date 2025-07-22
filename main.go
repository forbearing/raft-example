package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"demo/store"

	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
)

const (
	DefaultHTTPAddr = "localhost:13000"
	DefaultRaftAddr = "localhost:14000"
)

var (
	raftDir  string
	raftAddr string
	httpAddr string
	joinAddr string
	nodeID   string
)

func init() {
	pflag.StringVarP(&raftAddr, "raddr", "r", DefaultRaftAddr, "Set Raft bind address")
	pflag.StringVarP(&httpAddr, "haddr", "h", DefaultHTTPAddr, "Set HTTP bind address")
	pflag.StringVarP(&joinAddr, "join", "j", "", "Set join address")
	pflag.StringVar(&nodeID, "id", "", "Node ID. If not set, same as Raft bind address")
	pflag.Parse()

	if pflag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "No Raft storage directory specified\n")
		os.Exit(1)
	}
	raftDir = pflag.Arg(0)
	if len(raftDir) == 0 {
		fmt.Fprintf(os.Stderr, "No Raft storage directory specified\n")
		os.Exit(1)
	}

	if err := os.MkdirAll(raftDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create raft dir %s: %s\n", raftDir, err)
		os.Exit(1)
	}
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	s := store.New(raftDir, raftAddr, false)
	// Only leader node should open the store, it will bootstrap cluster.
	if err := s.Open(joinAddr == "", nodeID); err != nil {
		panic(err)
	}

	storeSvc := &service{s}

	// join the cluster
	join(joinAddr, raftAddr, nodeID)

	r.POST("/store", storeSvc.Set)
	r.GET("/store/:key", storeSvc.Get)
	r.POST("/join", storeSvc.Join)
	r.GET("/status", storeSvc.Status)

	if r.Run(httpAddr) != nil {
		panic(r.Run(httpAddr))
	}
}

func join(jAddr, rAddr, id string) error {
	if jAddr == "" {
		return nil
	}

	b, err := json.Marshal(JoinReq{ID: id, Addr: rAddr})
	if err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("http://%s/join", jAddr), "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

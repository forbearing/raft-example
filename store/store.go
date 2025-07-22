package store

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"demo/types"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	"github.com/sirupsen/logrus"
)

const (
	retainSnapshotCount = 2
	raftTimeout         = 10 * time.Second
)

type command struct {
	Op    string `json:"op,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type Store struct {
	raftDir  string
	raftAddr string
	raft     *raft.Raft
	inmem    bool

	m sync.Map // key type is string, value type is string.
}

func New(raftDir string, raftAddr string, inmem bool) *Store {
	return &Store{
		raftDir:  raftDir,
		inmem:    inmem,
		raftAddr: raftAddr,
	}
}

func (s *Store) Open(enableSingle bool, localID string) error {
	conf := raft.DefaultConfig()
	conf.LocalID = raft.ServerID(localID)

	addr, err := net.ResolveTCPAddr("tcp", s.raftAddr)
	if err != nil {
		return err
	}
	transport, err := raft.NewTCPTransport(s.raftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return err
	}

	snapshot, err := raft.NewFileSnapshotStore(s.raftDir, retainSnapshotCount, os.Stderr)
	if err != nil {
		return fmt.Errorf("file snapshot store: %s", err)
	}
	var logStore raft.LogStore
	var stableStore raft.StableStore
	if s.inmem {
		logStore = raft.NewInmemStore()
		stableStore = raft.NewInmemStore()
	} else {
		boltDB, err := raftboltdb.New(raftboltdb.Options{
			Path: filepath.Join(s.raftDir, "raft.db"),
		})
		if err != nil {
			return fmt.Errorf("new bbolt store: %s", err)
		}
		logStore = boltDB
		stableStore = boltDB
	}

	ra, err := raft.NewRaft(conf, (*fsm)(s), logStore, stableStore, snapshot, transport)
	if err != nil {
		return fmt.Errorf("new raft: %s", err)
	}
	s.raft = ra

	if enableSingle {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      conf.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		ra.BootstrapCluster(configuration)
	}

	return nil
}

func (s *Store) Get(key string) (string, error) {
	value, ok := s.m.Load(key)
	if !ok {
		return "", nil
	}
	return value.(string), nil
}

func (s *Store) Set(key, value string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}

	c := &command{
		Op:    "set",
		Key:   key,
		Value: value,
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return s.raft.Apply(b, raftTimeout).Error()
}

func (s *Store) Delete(key string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}

	c := &command{
		Op:  "delete",
		Key: key,
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return s.raft.Apply(b, raftTimeout).Error()
}

// Join joins a node, identified by nodeID and located at addr, to this store.
// The node must be ready to respond to Raft communications at that address.
func (s *Store) Join(nodeID, addr string) error {
	logrus.Infof("received join request for remote node %s at %s", nodeID, addr)

	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		logrus.Errorf("failed to get raft configuration: %v", err)
		return err
	}

	for _, srv := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if srv.Address == raft.ServerAddress(addr) && srv.ID == raft.ServerID(nodeID) {
				logrus.Infof("node %s at %s already member of cluster, ignoring join request", nodeID, addr)
				return nil
			}

			future := s.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, addr, err)
			}
		}
	}

	f := s.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if f.Error() != nil {
		return f.Error()
	}
	logrus.Infof("node %s at %s joined successfully", nodeID, addr)
	return nil
}

func (s *Store) Status() (types.StoreStatus, error) {
	leaderServerAddr, leaderId := s.raft.LeaderWithID()
	leader := types.Node{
		ID:      string(leaderId),
		Address: string(leaderServerAddr),
	}

	servers := s.raft.GetConfiguration().Configuration().Servers
	followers := []types.Node{}
	me := types.Node{
		Address: s.raftAddr,
	}

	for _, server := range servers {
		if server.ID != leaderId {
			followers = append(followers, types.Node{
				ID:      string(server.ID),
				Address: string(server.Address),
			})
		}
		if string(server.Address) == s.raftAddr {
			me = types.Node{
				ID:      string(server.ID),
				Address: string(server.Address),
			}
		}
	}

	status := types.StoreStatus{
		Me:        me,
		Leader:    leader,
		Followers: followers,
	}

	return status, nil
}

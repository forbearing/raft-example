package types

// Store is the interface Raft-backed key-value stores must implement.
type Store interface {
	// Get returns the value for the given key.
	Get(key string) (string, error)

	// Set sets the value for the given key, via distributed consensus.
	Set(key, value string) error

	// Delete removes the given key, via distributed consensus.
	Delete(key string) error

	// Join joins the node, identitifed by nodeID and reachable at addr, to the cluster.
	Join(nodeID string, addr string) error

	// Show who is me, the leader, and followers
	Status() (StoreStatus, error)
}

// StoreStatus is the Status a Store returns.
type StoreStatus struct {
	Me        Node   `json:"me"`
	Leader    Node   `json:"leader"`
	Followers []Node `json:"followers"`
}

// Node represents a node in the cluster.
type Node struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

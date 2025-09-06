package chain

import (
	"crypto/sha1"
	"fmt"
	"hash"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Node represents a physical or virtual machine in the network.
type Node struct {
	ID string
}

// DApp represents a decentralized application with a specific computational load.
// Its load is now dynamic, measured by the number of transactions it processes.
type DApp struct {
	ID                    string
	mu                    sync.Mutex
	ProcessedTransactions int // Simulates the computational load
}

// ProcessTransaction simulates a new transaction being processed by the DApp.
func (d *DApp) ProcessTransaction() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ProcessedTransactions++
}

// Shard represents a logical partition of the distributed ledger.
// It is maintained by a set of nodes for redundancy and hosts a collection of DApps.
type Shard struct {
	ID      string
	mu      sync.RWMutex
	dapps   map[string]*DApp // Maps DApp ID to the DApp
	nodeIDs []string         // The nodes responsible for this shard
}

// NewShard creates and initializes a new Shard.
func NewShard(id string, nodeIDs []string) *Shard {
	return &Shard{
		ID:      id,
		dapps:   make(map[string]*DApp),
		nodeIDs: nodeIDs,
	}
}

// AddDApp assigns a new DApp to the shard's state.
func (s *Shard) AddDApp(dapp *DApp) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dapps[dapp.ID] = dapp
	fmt.Printf("[Shard %s] Added DApp: %s\n", s.ID, dapp.ID)
}

// RemoveDApp removes a DApp from the shard's state.
func (s *Shard) RemoveDApp(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.dapps, id)
	fmt.Printf("[Shard %s] Removed DApp: %s\n", s.ID, id)
}

// GetDApp retrieves a DApp from the shard's state.
func (s *Shard) GetDApp(id string) (*DApp, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dapp, ok := s.dapps[id]
	return dapp, ok
}

// DappsDump returns a copy of the shard's DApp collection.
func (s *Shard) DappsDump() []*DApp {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dump := make([]*DApp, 0, len(s.dapps))
	for _, v := range s.dapps {
		dump = append(dump, v)
	}
	return dump
}

// ConsistentHasher is the core component for dynamic sharding.
// It maps keys (DApp IDs) to shards, minimizing rebalancing when shards are added or removed.
type ConsistentHasher struct {
	hash     hash.Hash
	replicas int
	mutex    sync.RWMutex
	keys     []int          // Sorted hash ring
	shards   map[int]string // Map from hash key to shard ID
}

// NewConsistentHasher creates a new ConsistentHasher.
func NewConsistentHasher(replicas int) *ConsistentHasher {
	return &ConsistentHasher{
		hash:     sha1.New(),
		replicas: replicas,
		shards:   make(map[int]string),
	}
}

// AddShard adds a new shard to the hash ring. It creates multiple virtual nodes for each shard
// to ensure a more even distribution.
func (ch *ConsistentHasher) AddShard(id string) {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	for i := 0; i < ch.replicas; i++ {
		hashKey := ch.hashKey(id + strconv.Itoa(i))
		ch.keys = append(ch.keys, hashKey)
		ch.shards[hashKey] = id
	}
	sort.Ints(ch.keys)
	fmt.Printf("ConsistentHasher: Added shard %s. Total keys on ring: %d\n", id, len(ch.keys))
}

// RemoveShard removes a shard from the hash ring and all its virtual nodes.
func (ch *ConsistentHasher) RemoveShard(id string) {
	ch.mutex.Lock()
	defer ch.mutex.Unlock()

	var newKeys []int
	for _, k := range ch.keys {
		if ch.shards[k] != id {
			newKeys = append(newKeys, k)
		} else {
			delete(ch.shards, k)
		}
	}
	ch.keys = newKeys
	sort.Ints(ch.keys)
	fmt.Printf("ConsistentHasher: Removed shard %s. Total keys on ring: %d\n", id, len(ch.keys))
}

// hashKey generates a SHA1 hash of a string and converts it to an integer.
func (ch *ConsistentHasher) hashKey(key string) int {
	ch.hash.Reset()
	ch.hash.Write([]byte(key))
	return int(ch.hash.Sum(nil)[0])
}

// GetShard determines which shard a key belongs to.
func (ch *ConsistentHasher) GetShard(key string) string {
	ch.mutex.RLock()
	defer ch.mutex.RUnlock()

	if len(ch.keys) == 0 {
		return ""
	}

	hashKey := ch.hashKey(key)

	// Binary search to find the shard on the ring.
	i := sort.Search(len(ch.keys), func(i int) bool { return ch.keys[i] >= hashKey })

	// If no key is found, wrap around to the beginning of the ring.
	if i == len(ch.keys) {
		i = 0
	}
	return ch.shards[ch.keys[i]]
}

// ShardManager orchestrates the distributed ledger network.
// It manages the collection of nodes and shards.
type ShardManager struct {
	nodes  map[string]*Node
	shards map[string]*Shard
	dapps  map[string]*DApp // Central registry of all DApps
	hasher *ConsistentHasher
	mu     sync.RWMutex

	// Parameters for automatic management
	maxShardLoad int
	minShardLoad int
	maxNodes     int
	minNodes     int
	shardCounter int // Used to generate unique IDs for new shards
	nodeCounter  int // Used to generate unique IDs for new nodes
}

// NewShardManager initializes a new network with an initial number of nodes and shards.
func NewShardManager(initialNodes, initialShards, maxShardLoad, minShardLoad, maxNodes, minNodes int) *ShardManager {

	rand.Seed(time.Now().UnixNano())

	manager := &ShardManager{
		nodes:        make(map[string]*Node),
		shards:       make(map[string]*Shard),
		dapps:        make(map[string]*DApp),
		hasher:       NewConsistentHasher(20),
		maxShardLoad: maxShardLoad,
		minShardLoad: minShardLoad,
		maxNodes:     maxNodes,
		minNodes:     minNodes,
		shardCounter: initialShards,
		nodeCounter:  initialNodes,
	}

	fmt.Println("Initializing distributed ledger network...")

	for i := 0; i < initialNodes; i++ {
		nodeID := fmt.Sprintf("node-%d", i+1)
		manager.AddNode(nodeID)
	}

	for i := 0; i < initialShards; i++ {
		shardID := fmt.Sprintf("shard-%d", i+1)
		manager.AddShard(shardID)
	}

	return manager
}

// AddNode adds a new node to the network and assigns it to a random shard.
func (sm *ShardManager) AddNode(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.nodes[id]; exists {
		fmt.Printf("Node %s already exists.\n", id)
		return
	}

	sm.nodes[id] = &Node{ID: id}

	if len(sm.shards) > 0 {
		var shardIDs []string
		for sid := range sm.shards {
			shardIDs = append(shardIDs, sid)
		}
		targetShardID := shardIDs[rand.Intn(len(shardIDs))]
		sm.shards[targetShardID].nodeIDs = append(sm.shards[targetShardID].nodeIDs, id)
		fmt.Printf("Added Node %s and assigned it to shard %s\n", id, targetShardID)
	} else {
		fmt.Printf("Added Node %s, but no shards exist to assign it to.\n", id)
	}
}

// RemoveNode simulates a node leaving the network and reassigns its shards.
func (sm *ShardManager) RemoveNode(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.nodes[id]; !exists {
		fmt.Printf("Node %s not found.\n", id)
		return
	}

	fmt.Printf("\n--- Node %s disconnected. Re-assigning its shards... ---\n", id)

	for sid, shard := range sm.shards {
		var newNodes []string
		for _, nid := range shard.nodeIDs {
			if nid != id {
				newNodes = append(newNodes, nid)
			}
		}
		sm.shards[sid].nodeIDs = newNodes
	}

	delete(sm.nodes, id)
}

func (sm *ShardManager) manageShards() {
	sm.mu.RLock()
	var shardIDs []string
	for id := range sm.shards {
		shardIDs = append(shardIDs, id)
	}
	sm.mu.RUnlock()

	for _, id := range shardIDs {
		sm.mu.RLock()
		shard, ok := sm.shards[id]
		sm.mu.RUnlock()
		if !ok {
			continue
		}

		shard.mu.RLock()
		totalLoad := 0
		for _, dapp := range shard.dapps {
			dapp.mu.Lock()
			totalLoad += dapp.ProcessedTransactions
			dapp.mu.Unlock()
		}
		shard.mu.RUnlock()

		if totalLoad > sm.maxShardLoad && len(sm.shards) < sm.maxNodes {
			sm.SplitShard(id)
		} else if totalLoad < sm.minShardLoad && len(sm.shards) > sm.minNodes {
			sm.MergeShard(id)
		}
	}
}

// AddShard adds a new logical shard to the network.
func (sm *ShardManager) AddShard(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.shards[id]; exists {
		fmt.Printf("Shard %s already exists.\n", id)
		return
	}

	var activeNodeIDs []string
	for nid := range sm.nodes {
		activeNodeIDs = append(activeNodeIDs, nid)
	}

	newShard := NewShard(id, activeNodeIDs)
	sm.shards[id] = newShard
	sm.hasher.AddShard(id)
	fmt.Printf("Added logical shard %s and assigned it to nodes: %v\n", id, newShard.nodeIDs)
}

// MergeShard simulates a shard being merged into others.
func (sm *ShardManager) MergeShard(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	shardToMerge, exists := sm.shards[id]
	if !exists {
		fmt.Printf("Shard %s not found.\n", id)
		return
	}

	fmt.Printf("\n--- Merging Shard %s (underutilized) ---\n", id)

	dappsToMigrate := shardToMerge.DappsDump()
	sm.hasher.RemoveShard(id)
	delete(sm.shards, id)

	for _, dapp := range dappsToMigrate {
		newShardID := sm.hasher.GetShard(dapp.ID)
		fmt.Printf("Migrating DApp '%s' to new shard '%s'\n", dapp.ID, newShardID)
		if newShard, ok := sm.shards[newShardID]; ok {
			newShard.AddDApp(dapp)
		}
	}
}

// SplitShard simulates an overloaded shard splitting its data.
func (sm *ShardManager) SplitShard(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	shardToSplit, exists := sm.shards[id]
	if !exists {
		fmt.Printf("Shard %s not found.\n", id)
		return
	}

	sm.shardCounter++
	newShardID := fmt.Sprintf("shard-%d", sm.shardCounter)

	fmt.Printf("\n--- Splitting Shard %s to create new shard %s ---\n", id, newShardID)

	sm.AddShard(newShardID)

	dappsToMigrate := shardToSplit.DappsDump()

	// Sort DApps by their load to ensure a more even split
	sort.Slice(dappsToMigrate, func(i, j int) bool {
		return dappsToMigrate[i].ProcessedTransactions > dappsToMigrate[j].ProcessedTransactions
	})

	halfLoad := 0
	for _, dapp := range dappsToMigrate {
		halfLoad += dapp.ProcessedTransactions
	}
	halfLoad /= 2

	var migratedLoad int
	for _, dapp := range dappsToMigrate {
		if migratedLoad < halfLoad {
			sm.mu.RLock()
			newShard, ok := sm.shards[newShardID]
			sm.mu.RUnlock()
			if ok {
				newShard.AddDApp(dapp)
				migratedLoad += dapp.ProcessedTransactions
				shardToSplit.RemoveDApp(dapp.ID)
			}
		}
	}
}

// DeployDapp routes a new DApp to the correct shard and deploys it.
func (sm *ShardManager) DeployDapp(dappID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	dapp := &DApp{ID: dappID}
	sm.dapps[dappID] = dapp
	sm.mu.RUnlock()

	shardID := sm.hasher.GetShard(dappID)
	shard, ok := sm.shards[shardID]
	sm.mu.RUnlock()

	if !ok {
		fmt.Printf("Error: Shard %s not found for DApp %s. Re-routing...\n", shardID, dappID)
		sm.DeployDapp(dappID)
		return
	}

	nodeID := shard.nodeIDs[rand.Intn(len(shard.nodeIDs))]
	fmt.Printf("Routing DApp '%s' to node %s on shard %s\n", dappID, nodeID, shard.ID)
	shard.AddDApp(dapp)
	sm.manageShards()
}

// ProcessDAppTransaction simulates a transaction for a given DApp.
func (sm *ShardManager) ProcessDAppTransaction(dappID string) {
	sm.mu.RLock()
	dapp, ok := sm.dapps[dappID]
	sm.mu.RUnlock()
	if !ok {
		fmt.Printf("Error: DApp %s not found.\n", dappID)
		return
	}
	dapp.ProcessTransaction()
	sm.manageShards()
}

// RemoveDapp API method to remove a DApp from the network.
func (sm *ShardManager) RemoveDapp(dappID string) {
	sm.mu.RLock()
	shardID := sm.hasher.GetShard(dappID)
	shard, ok := sm.shards[shardID]
	sm.mu.RUnlock()

	if !ok {
		fmt.Printf("Error: Shard %s not found for DApp %s. Cannot remove.\n", shardID, dappID)
		return
	}
	shard.RemoveDApp(dappID)

	sm.mu.Lock()
	delete(sm.dapps, dappID)
	sm.mu.Unlock()
	sm.manageShards()
}

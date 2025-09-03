package chain

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
)

type SmartContract struct {
	ID               string
	ShardID          int64
	TransactionCount int64
}

type Node struct {
	ID      string
	ShardID int64
	Power   int
}

type Shard struct {
	ID        int64
	Contracts []SmartContract
	Nodes     []Node
	Capacity  int
	Load      int64
}

type DynamicShardingSystem struct {
	Shards        []Shard
	Nodes         []Node
	chain         *Chain
	pendingMerges *cmap.ConcurrentMap[string, int64]
	contracts     *cmap.ConcurrentMap[string, *SmartContract]
}

func (ds *DynamicShardingSystem) AssignContract(sc SmartContract) {
	if len(ds.Shards) == 0 {
		fmt.Println("No shards exist. Creating initial shard...")
		ds.CreateNewShard()
	}

	var selectedShard *Shard
	for i := range ds.Shards {
		if len(ds.Shards[i].Contracts) < ds.Shards[i].Capacity {
			selectedShard = &ds.Shards[i]
			break
		}
	}

	if selectedShard == nil {
		selectedShard = ds.CreateNewShard()
	}

	sc.ShardID = selectedShard.ID
	selectedShard.Contracts = append(selectedShard.Contracts, sc)
	selectedShard.Load += sc.TransactionCount
	fmt.Printf("Contract %s assigned to Shard %d\n", sc.ID, selectedShard.ID)
}

func (ds *DynamicShardingSystem) CreateNewShard() *Shard {
	newShard := Shard{ID: int64(len(ds.Shards) + 1), Capacity: 10, Load: 0, Contracts: []SmartContract{}, Nodes: []Node{}}
	ds.Shards = append(ds.Shards, newShard)
	fmt.Printf("Created new Shard %d\n", newShard.ID)
	ds.chain.CreateShardChain(newShard.ID)
	return &newShard
}

func (ds *DynamicShardingSystem) HandleNewNode(newNode Node) {

	if len(ds.Shards) == 0 {
		fmt.Println("No shards exist. Creating first shard...")
		ds.CreateNewShard()
	}

	var bestShard *Shard
	for i := range ds.Shards {
		if bestShard == nil || ds.Shards[i].Load < bestShard.Load {
			bestShard = &ds.Shards[i]
		}
	}

	newNode.ShardID = bestShard.ID
	ds.CheckAndModifyMyShards(newNode.ID, newNode.ShardID)
	bestShard.Nodes = append(bestShard.Nodes, newNode)
	ds.Nodes = append(ds.Nodes, newNode)
	c, _ := ds.chain.blockchain.allSubChains.Get(fmt.Sprintf("%d", bestShard.ID))
	c.peers[newNode.ID] = 100
	fmt.Printf("New Node %s assigned to Shard %d\n", newNode.ID, bestShard.ID)
}

func (ds *DynamicShardingSystem) LogLoad(shardId int64, machineId string) {
	for i := range ds.Shards {
		if ds.Shards[i].ID == shardId {
			for j := range ds.Shards[i].Contracts {
				if ds.Shards[i].Contracts[j].ID == machineId {
					ds.Shards[i].Contracts[j].TransactionCount += 1
					ds.Shards[i].Load += 1
					break
				}
			}
			break
		}
	}
	ds.CheckAndSplitShards()
}

func (ds *DynamicShardingSystem) CheckAndSplitShards() {
	for i := range ds.Shards {
		if len(ds.Shards[i].Contracts) > 10 {
			fmt.Printf("Shard %d overloaded! Splitting into two balanced shards...\n", ds.Shards[i].ID)
			ds.SplitShardSmartly(&ds.Shards[i])
		}
	}
}

func (ds *DynamicShardingSystem) CheckAndModifyMyShards(nodeId string, shardId int64) {
	ds.chain.MyShards.Clear()
	if !ds.chain.MyShards.Has(fmt.Sprintf("%d", shardId)) {
		ds.chain.MyShards.Set(fmt.Sprintf("%d", shardId), true)
		shardChain, _ := ds.chain.blockchain.allSubChains.Get(fmt.Sprintf("%d", shardId))
		shardChain.Run()
	}
}

func (ds *DynamicShardingSystem) SplitShardSmartly(shard *Shard) {
	sort.Slice(shard.Contracts, func(i, j int) bool {
		return shard.Contracts[i].TransactionCount > shard.Contracts[j].TransactionCount
	})

	shardA := ds.CreateNewShard()
	shardB := ds.CreateNewShard()

	loadA, loadB := int64(0), int64(0)
	for _, contract := range shard.Contracts {
		if loadA <= loadB {
			contract.ShardID = shardA.ID
			shardA.Contracts = append(shardA.Contracts, contract)
			loadA += contract.TransactionCount
		} else {
			contract.ShardID = shardB.ID
			shardB.Contracts = append(shardB.Contracts, contract)
			loadB += contract.TransactionCount
		}
	}

	for _, node := range shard.Nodes {
		if len(shardA.Nodes) <= len(shardB.Nodes) {
			node.ShardID = shardA.ID
			shardA.Nodes = append(shardA.Nodes, node)
			ds.CheckAndModifyMyShards(node.ID, node.ShardID)
		} else {
			node.ShardID = shardB.ID
			shardB.Nodes = append(shardB.Nodes, node)
			ds.CheckAndModifyMyShards(node.ID, node.ShardID)
		}
	}

	fmt.Printf("Shard %d removed, contracts redistributed to %d (Load: %d) and %d (Load: %d)\n",
		shard.ID, shardA.ID, loadA, shardB.ID, loadB)
	fmt.Printf("Nodes redistributed: Shard %d -> %d nodes, Shard %d -> %d nodes\n",
		shardA.ID, len(shardA.Nodes), shardB.ID, len(shardB.Nodes))

	ds.RemoveShard(shard.ID)
}

func (ds *DynamicShardingSystem) CheckAndMergeShards() {
	if len(ds.Shards) < 2 {
		return
	}

	shardA := &ds.Shards[len(ds.Shards)-2]
	shardB := &ds.Shards[len(ds.Shards)-1]

	fmt.Printf("Merging Shard %d and Shard %d...\n", shardA.ID, shardB.ID)

	ds.IncrementalMerge(shardA, shardB)

	ds.RemoveShard(shardB.ID)
}

func (ds *DynamicShardingSystem) DoPostMerge(shardBId int64) {
	shardAId, _ := ds.pendingMerges.Get(fmt.Sprintf("%d", shardBId))
	var shardA, shardB *Shard
	for _, shard := range ds.Shards {
		if shard.ID == shardAId {
			shardA = &shard
		} else if shard.ID == shardBId {
			shardB = &shard
		}
		if shardA != nil && shardB != nil {
			break
		}
	}
	ds.TryMerge(shardA, shardB)

	shardA.Contracts = append(shardA.Contracts, shardB.Contracts...)
	shardA.Load += ds.CalculateLoad(shardB.Contracts)
	fmt.Printf("Transferred batch of %d contracts to Shard %d\n", len(shardB.Contracts), shardA.ID)

	shardBEvents, _ := ds.chain.SubChains.Get(fmt.Sprintf("%d", shardB.ID))
	pendingTrxsLeft, _ := json.Marshal(shardBEvents.pendingTrxs)
	b := make([]byte, 1+4+len(pendingTrxsLeft))
	b[0] = 0xa2
	bLen := make([]byte, 4)
	binary.LittleEndian.PutUint32(bLen, uint32(len(pendingTrxsLeft)))
	copy(b[1:5], bLen)
	copy(b[5:], pendingTrxsLeft)
	shardAEvents, _ := ds.chain.SubChains.Get(fmt.Sprintf("%d", shardA.ID))
	shardAEvents.BroadcastInShard(b)

	fmt.Printf("Shard %d successfully merged with Shard %d!\n", shardA.ID, shardB.ID)
}

func (ds *DynamicShardingSystem) TryMerge(shardA, shardB *Shard) {
	if ds.chain.MyShards.Has(fmt.Sprintf("%d", shardA.ID)) && ds.chain.MyShards.Has(fmt.Sprintf("%d", shardB.ID)) {
		func() {
			shardAEvents, _ := ds.chain.SubChains.Get(fmt.Sprintf("%d", shardA.ID))
			shardBEvents, _ := ds.chain.SubChains.Get(fmt.Sprintf("%d", shardB.ID))
			shardAEvents.Lock.Lock()
			defer shardAEvents.Lock.Unlock()
			shardBEvents.Lock.Lock()
			defer shardBEvents.Lock.Unlock()
			shardBEvents.removed = true
			events := []*Event{}
			aI := 0
			bI := 0
			for (aI < len(shardAEvents.blocks)) || (bI < len(shardBEvents.blocks)) {
				if len(shardAEvents.blocks) <= aI {
					events = append(events, shardBEvents.blocks...)
					bI = len(shardBEvents.blocks)
				} else if len(shardBEvents.blocks) <= bI {
					events = append(events, shardAEvents.blocks...)
					aI = len(shardAEvents.blocks)
				} else if shardAEvents.blocks[aI].Timestamp < shardBEvents.blocks[bI].Timestamp {
					events = append(events, shardAEvents.blocks[aI])
					aI++
				} else {
					events = append(events, shardBEvents.blocks[bI])
					bI++
				}
			}
			shardAEvents.blocks = events
			sendingEvents := shardBEvents.blocks
			eventsData, _ := json.Marshal(sendingEvents)
			b := make([]byte, 1+4+len(eventsData))
			b[0] = 0xa1
			bLen := make([]byte, 4)
			binary.LittleEndian.PutUint32(bLen, uint32(len(eventsData)))
			copy(b[1:5], bLen)
			copy(b[5:], eventsData)
			shardAEvents.BroadcastInShard(b)
			for _, e := range shardBEvents.events {
				for _, trx := range e.Transactions {
					if trx.Typ == "response" {
						ds.chain.blockchain.pipeline([][]byte{trx.Payload})
					}
				}
			}
		}()
	} else if ds.chain.MyShards.Has(fmt.Sprintf("%d", shardB.ID)) {
		func() {
			shardAEvents, _ := ds.chain.SubChains.Get(fmt.Sprintf("%d", shardA.ID))
			shardBEvents, _ := ds.chain.SubChains.Get(fmt.Sprintf("%d", shardB.ID))
			shardBEvents.Lock.Lock()
			defer shardBEvents.Lock.Unlock()
			shardBEvents.removed = true
			events := shardBEvents.blocks
			eventsData, _ := json.Marshal(events)
			b := make([]byte, 1+4+len(eventsData))
			b[0] = 0xa1
			bLen := make([]byte, 4)
			binary.LittleEndian.PutUint32(bLen, uint32(len(eventsData)))
			copy(b[1:5], bLen)
			copy(b[5:], eventsData)
			shardAEvents.BroadcastInShard(b)
		}()
	}
}

func (ds *DynamicShardingSystem) IncrementalMerge(shardA, shardB *Shard) {

	ds.TryMerge(shardA, shardB)

	shardA.Nodes = append(shardA.Nodes, shardB.Nodes...)
	fmt.Printf("Transferred batch of %d nodes to Shard %d\n", len(shardB.Nodes), shardA.ID)
}

func (ds *DynamicShardingSystem) CalculateLoad(contracts []SmartContract) int64 {
	totalLoad := int64(0)
	for _, contract := range contracts {
		totalLoad += contract.TransactionCount
	}
	return totalLoad
}

func (ds *DynamicShardingSystem) RemoveShard(shardID int64) {
	for i, shard := range ds.Shards {
		if shard.ID == shardID {
			ds.Shards = append(ds.Shards[:i], ds.Shards[i+1:]...)
			break
		}
	}
	ds.chain.MyShards.Remove(fmt.Sprintf("%d", shardID))
}

func NewSharder(chain *Chain) *DynamicShardingSystem {
	rand.Seed(time.Now().UnixNano())
	m := cmap.New[int64]()
	m2 := cmap.New[*SmartContract]()
	return &DynamicShardingSystem{chain: chain, pendingMerges: &m, contracts: &m2, Shards: []Shard{}, Nodes: []Node{}}
}

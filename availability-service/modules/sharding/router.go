package sharding

import (
	"fmt"
	"hash/crc32"
	"sort"
)

// ShardRoute describes where a logical partition is physically stored.
type ShardRoute struct {
	Partition int
	Shard     string
}

// Router maps stable business keys to logical partitions and then to shards.
type Router struct {
	partitionCount int
	partitionMap   map[int]string
}

// NewRouter creates a router with a fixed number of logical partitions.
// Every partition in [0, partitionCount) must be assigned to a non-empty shard.
func NewRouter(partitionCount int, partitionMap map[int]string) (*Router, error) {
	if partitionCount <= 0 {
		return nil, fmt.Errorf("partition count must be greater than zero")
	}

	partitions := make(map[int]string, len(partitionMap))
	for partition := 0; partition < partitionCount; partition++ {
		shard, ok := partitionMap[partition]
		if !ok {
			return nil, fmt.Errorf("partition %d has no shard configured", partition)
		}
		if shard == "" {
			return nil, fmt.Errorf("partition %d has empty shard configured", partition)
		}
		partitions[partition] = shard
	}

	return &Router{
		partitionCount: partitionCount,
		partitionMap:   partitions,
	}, nil
}

// ShardForDoctor maps a doctor_id to one logical partition using CRC32.
func (r *Router) ShardForDoctor(doctorID string) (ShardRoute, error) {
	if doctorID == "" {
		return ShardRoute{}, fmt.Errorf("doctor_id cannot be empty")
	}

	checksum := crc32.ChecksumIEEE([]byte(doctorID))
	partition := int(uint64(checksum) % uint64(r.partitionCount))

	return r.ShardForPartition(partition)
}

// ShardForPartition returns the shard assigned to an existing logical partition.
func (r *Router) ShardForPartition(partition int) (ShardRoute, error) {
	if partition < 0 || partition >= r.partitionCount {
		return ShardRoute{}, fmt.Errorf("partition %d out of range", partition)
	}

	shard, ok := r.partitionMap[partition]
	if !ok || shard == "" {
		return ShardRoute{}, fmt.Errorf("partition %d has no shard configured", partition)
	}

	return ShardRoute{Partition: partition, Shard: shard}, nil
}

// AllShards returns the unique configured shard names in deterministic order.
func (r *Router) AllShards() []string {
	seen := make(map[string]struct{}, len(r.partitionMap))
	for _, shard := range r.partitionMap {
		seen[shard] = struct{}{}
	}

	shards := make([]string, 0, len(seen))
	for shard := range seen {
		shards = append(shards, shard)
	}
	sort.Strings(shards)

	return shards
}

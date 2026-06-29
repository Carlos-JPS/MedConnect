package sharding

import (
	"reflect"
	"testing"
)

func TestShardForDoctorIsDeterministic(t *testing.T) {
	router := newTestRouter(t)

	first, err := router.ShardForDoctor("doctor-123")
	if err != nil {
		t.Fatalf("ShardForDoctor returned error: %v", err)
	}

	second, err := router.ShardForDoctor("doctor-123")
	if err != nil {
		t.Fatalf("ShardForDoctor returned error: %v", err)
	}

	if first != second {
		t.Fatalf("expected deterministic route, got %+v and %+v", first, second)
	}
}

func TestShardForDoctorRejectsEmptyDoctorID(t *testing.T) {
	router := newTestRouter(t)

	if _, err := router.ShardForDoctor(""); err == nil {
		t.Fatal("expected error for empty doctor_id")
	}
}

func TestShardForPartitionReturnsConfiguredShard(t *testing.T) {
	router := newTestRouter(t)

	route, err := router.ShardForPartition(2)
	if err != nil {
		t.Fatalf("ShardForPartition returned error: %v", err)
	}

	want := ShardRoute{Partition: 2, Shard: "shard-b"}
	if route != want {
		t.Fatalf("expected %+v, got %+v", want, route)
	}
}

func TestShardForPartitionRejectsOutOfRangePartition(t *testing.T) {
	router := newTestRouter(t)

	for _, partition := range []int{-1, 4} {
		if _, err := router.ShardForPartition(partition); err == nil {
			t.Fatalf("expected error for partition %d", partition)
		}
	}
}

func TestNewRouterRejectsInvalidPartitionCount(t *testing.T) {
	if _, err := NewRouter(0, map[int]string{}); err == nil {
		t.Fatal("expected error for zero partition count")
	}

	if _, err := NewRouter(-1, map[int]string{}); err == nil {
		t.Fatal("expected error for negative partition count")
	}
}

func TestNewRouterRequiresAllPartitions(t *testing.T) {
	partitionMap := map[int]string{
		0: "shard-a",
		2: "shard-b",
	}

	if _, err := NewRouter(3, partitionMap); err == nil {
		t.Fatal("expected error for missing partition")
	}
}

func TestNewRouterRejectsEmptyShard(t *testing.T) {
	partitionMap := map[int]string{
		0: "shard-a",
		1: "",
	}

	if _, err := NewRouter(2, partitionMap); err == nil {
		t.Fatal("expected error for empty shard")
	}
}

func TestAllShardsReturnsSortedUniqueShards(t *testing.T) {
	partitionMap := map[int]string{
		0: "shard-b",
		1: "shard-a",
		2: "shard-b",
		3: "shard-c",
	}
	router, err := NewRouter(4, partitionMap)
	if err != nil {
		t.Fatalf("NewRouter returned error: %v", err)
	}

	got := router.AllShards()
	want := []string{"shard-a", "shard-b", "shard-c"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestRouterCopiesPartitionMap(t *testing.T) {
	partitionMap := map[int]string{
		0: "shard-a",
		1: "shard-b",
	}
	router, err := NewRouter(2, partitionMap)
	if err != nil {
		t.Fatalf("NewRouter returned error: %v", err)
	}

	partitionMap[1] = "shard-c"

	route, err := router.ShardForPartition(1)
	if err != nil {
		t.Fatalf("ShardForPartition returned error: %v", err)
	}

	if route.Shard != "shard-b" {
		t.Fatalf("expected copied shard-b, got %q", route.Shard)
	}
}

func newTestRouter(t *testing.T) *Router {
	t.Helper()

	router, err := NewRouter(4, map[int]string{
		0: "shard-a",
		1: "shard-a",
		2: "shard-b",
		3: "shard-c",
	})
	if err != nil {
		t.Fatalf("NewRouter returned error: %v", err)
	}

	return router
}

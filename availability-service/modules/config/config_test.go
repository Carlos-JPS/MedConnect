package config

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestLoadSingleDBModeKeepsDSNAndDisablesShardingByDefault(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("DB_HOST", "availability-db")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "postgres")
	t.Setenv("DB_NAME", "availability_db")
	t.Setenv("GRPC_PORT", "50051")

	cfg := Load()

	if cfg.ShardingEnabled {
		t.Fatal("expected sharding disabled when AVAILABILITY_SHARDING_ENABLED is absent")
	}
	if cfg.PartitionCount != 0 {
		t.Fatalf("expected no partition count in single DB mode, got %d", cfg.PartitionCount)
	}
	if len(cfg.Shards) != 0 || len(cfg.PartitionMap) != 0 || len(cfg.ShardDSNs) != 0 {
		t.Fatalf("expected empty sharding fields, got shards=%v partitionMap=%v shardDSNs=%v", cfg.Shards, cfg.PartitionMap, cfg.ShardDSNs)
	}

	wantDSN := "host=availability-db port=5432 user=postgres password=postgres dbname=availability_db sslmode=disable"
	if got := cfg.DSN(); got != wantDSN {
		t.Fatalf("expected DSN %q, got %q", wantDSN, got)
	}
	if err := cfg.ValidateShardingConfig(); err != nil {
		t.Fatalf("ValidateShardingConfig returned error in single DB mode: %v", err)
	}
}

func TestLoadValidShardedMode(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("AVAILABILITY_SHARDING_ENABLED", "true")
	t.Setenv("AVAILABILITY_PARTITION_COUNT", "4")
	t.Setenv("AVAILABILITY_SHARDS", "shard0, shard1")
	t.Setenv("AVAILABILITY_PARTITION_MAP", "0:shard0,1:shard1,2:shard0,3:shard1")
	t.Setenv("AVAILABILITY_SHARD0_DSN", "postgres://user:pass@availability-shard0:5432/availability_db?sslmode=disable")
	t.Setenv("AVAILABILITY_SHARD1_DSN", "postgres://user:pass@availability-shard1:5432/availability_db?sslmode=disable")

	cfg := Load()

	if !cfg.ShardingEnabled {
		t.Fatal("expected sharding enabled")
	}
	if cfg.PartitionCount != 4 {
		t.Fatalf("expected partition count 4, got %d", cfg.PartitionCount)
	}
	if !reflect.DeepEqual(cfg.Shards, []string{"shard0", "shard1"}) {
		t.Fatalf("unexpected shards: %v", cfg.Shards)
	}
	wantMap := map[int]string{0: "shard0", 1: "shard1", 2: "shard0", 3: "shard1"}
	if !reflect.DeepEqual(cfg.PartitionMap, wantMap) {
		t.Fatalf("expected partition map %v, got %v", wantMap, cfg.PartitionMap)
	}
	if cfg.ShardDSNs["shard0"] == "" || cfg.ShardDSNs["shard1"] == "" {
		t.Fatalf("expected DSNs for both shards, got %v", cfg.ShardDSNs)
	}
	if err := cfg.ValidateShardingConfig(); err != nil {
		t.Fatalf("ValidateShardingConfig returned error: %v", err)
	}
}

func TestLoadShardedModeDefaultsPartitionCountWhenEnabled(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("AVAILABILITY_SHARDING_ENABLED", "yes")
	t.Setenv("AVAILABILITY_SHARDS", "shard0")
	t.Setenv("AVAILABILITY_PARTITION_MAP", partitionsToSingleShard(16, "shard0"))
	t.Setenv("AVAILABILITY_SHARD0_DSN", "postgres://user:pass@availability-shard0:5432/availability_db?sslmode=disable")

	cfg := Load()

	if cfg.PartitionCount != 16 {
		t.Fatalf("expected default partition count 16 when sharding is enabled, got %d", cfg.PartitionCount)
	}
	if err := cfg.ValidateShardingConfig(); err != nil {
		t.Fatalf("ValidateShardingConfig returned error: %v", err)
	}
}

func TestValidateShardingConfigRejectsIncompletePartitionMap(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("AVAILABILITY_SHARDING_ENABLED", "true")
	t.Setenv("AVAILABILITY_PARTITION_COUNT", "2")
	t.Setenv("AVAILABILITY_SHARDS", "shard0")
	t.Setenv("AVAILABILITY_PARTITION_MAP", "0:shard0")
	t.Setenv("AVAILABILITY_SHARD0_DSN", "postgres://user:pass@availability-shard0:5432/availability_db?sslmode=disable")

	cfg := Load()

	assertErrorContains(t, cfg.ValidateShardingConfig(), "partition 1 has no shard configured")
}

func TestValidateShardingConfigRejectsMissingShardDSN(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("AVAILABILITY_SHARDING_ENABLED", "true")
	t.Setenv("AVAILABILITY_PARTITION_COUNT", "1")
	t.Setenv("AVAILABILITY_SHARDS", "shard0")
	t.Setenv("AVAILABILITY_PARTITION_MAP", "0:shard0")

	cfg := Load()

	assertErrorContains(t, cfg.ValidateShardingConfig(), "AVAILABILITY_SHARD0_DSN")
}

func TestValidateShardingConfigRejectsInvalidBool(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("AVAILABILITY_SHARDING_ENABLED", "sometimes")

	cfg := Load()

	assertErrorContains(t, cfg.ValidateShardingConfig(), "invalid AVAILABILITY_SHARDING_ENABLED")
}

func TestValidateShardingConfigRejectsInvalidPartitionCount(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("AVAILABILITY_SHARDING_ENABLED", "true")
	t.Setenv("AVAILABILITY_PARTITION_COUNT", "0")
	t.Setenv("AVAILABILITY_SHARDS", "shard0")
	t.Setenv("AVAILABILITY_PARTITION_MAP", "0:shard0")
	t.Setenv("AVAILABILITY_SHARD0_DSN", "postgres://user:pass@availability-shard0:5432/availability_db?sslmode=disable")

	cfg := Load()

	assertErrorContains(t, cfg.ValidateShardingConfig(), "invalid AVAILABILITY_PARTITION_COUNT")
}

func withCleanEnv(t *testing.T) {
	t.Helper()

	original := os.Environ()
	os.Clearenv()
	t.Cleanup(func() {
		os.Clearenv()
		for _, entry := range original {
			key, value, found := strings.Cut(entry, "=")
			if found {
				_ = os.Setenv(key, value)
			}
		}
	})
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %v", want, err)
	}
}

func partitionsToSingleShard(count int, shard string) string {
	entries := make([]string, 0, count)
	for partition := 0; partition < count; partition++ {
		entries = append(entries, strings.Join([]string{strconv.Itoa(partition), shard}, ":"))
	}

	return strings.Join(entries, ",")
}

package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/Carlos-JPS/medconnect/availability-service/modules/sharding"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	GRPCPort   string

	ShardingEnabled bool
	PartitionCount  int
	Shards          []string
	ShardDSNs       map[string]string
	PartitionMap    map[int]string

	loadErrors []error
}

func Load() *Config {
	cfg := &Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		GRPCPort:   os.Getenv("GRPC_PORT"),
	}

	cfg.loadSharding(os.LookupEnv)

	return cfg
}

func (c *Config) DSN() string {
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=disable"
}

// ValidateShardingConfig checks only the optional sharding block.
// Single-DB mode remains valid when sharding is disabled, preserving DSN().
func (c *Config) ValidateShardingConfig() error {
	if c == nil {
		return errors.New("config cannot be nil")
	}

	errs := append([]error{}, c.loadErrors...)
	if !c.ShardingEnabled {
		return errors.Join(errs...)
	}

	if c.PartitionCount <= 0 {
		errs = append(errs, errors.New("AVAILABILITY_PARTITION_COUNT must be greater than zero when sharding is enabled"))
	}

	declaredShards := make(map[string]struct{}, len(c.Shards))
	for _, shard := range c.Shards {
		shard = strings.TrimSpace(shard)
		if shard == "" {
			errs = append(errs, errors.New("AVAILABILITY_SHARDS cannot contain empty shard names"))
			continue
		}
		if _, exists := declaredShards[shard]; exists {
			errs = append(errs, fmt.Errorf("AVAILABILITY_SHARDS contains duplicate shard %q", shard))
			continue
		}
		declaredShards[shard] = struct{}{}
	}
	if len(declaredShards) == 0 {
		errs = append(errs, errors.New("AVAILABILITY_SHARDS must declare at least one shard when sharding is enabled"))
	}

	for shard := range declaredShards {
		if strings.TrimSpace(c.ShardDSNs[shard]) == "" {
			errs = append(errs, fmt.Errorf("missing DSN for shard %q; set %s", shard, shardDSNEnvName(shard)))
		}
	}

	for partition, shard := range c.PartitionMap {
		if partition < 0 || partition >= c.PartitionCount {
			errs = append(errs, fmt.Errorf("partition %d is outside configured range [0,%d)", partition, c.PartitionCount))
		}
		if _, ok := declaredShards[shard]; !ok {
			errs = append(errs, fmt.Errorf("partition %d references undeclared shard %q", partition, shard))
		}
	}

	router, err := sharding.NewRouter(c.PartitionCount, c.PartitionMap)
	if err != nil {
		errs = append(errs, fmt.Errorf("invalid partition map: %w", err))
	} else {
		for _, shard := range router.AllShards() {
			if _, ok := declaredShards[shard]; !ok {
				errs = append(errs, fmt.Errorf("partition map uses undeclared shard %q", shard))
			}
			if strings.TrimSpace(c.ShardDSNs[shard]) == "" {
				errs = append(errs, fmt.Errorf("missing DSN for routed shard %q; set %s", shard, shardDSNEnvName(shard)))
			}
		}
	}

	return errors.Join(errs...)
}

func (c *Config) loadSharding(lookupEnv func(string) (string, bool)) {
	if raw, ok := lookupEnv("AVAILABILITY_SHARDING_ENABLED"); ok {
		parsed, err := parseBool(raw)
		if err != nil {
			c.loadErrors = append(c.loadErrors, fmt.Errorf("invalid AVAILABILITY_SHARDING_ENABLED: %w", err))
		} else {
			c.ShardingEnabled = parsed
		}
	}

	if raw, ok := lookupEnv("AVAILABILITY_PARTITION_COUNT"); ok {
		parsed, err := parsePositiveInt(raw)
		if err != nil {
			c.loadErrors = append(c.loadErrors, fmt.Errorf("invalid AVAILABILITY_PARTITION_COUNT: %w", err))
		} else {
			c.PartitionCount = parsed
		}
	} else if c.ShardingEnabled {
		c.PartitionCount = 16
	}

	if raw, ok := lookupEnv("AVAILABILITY_SHARDS"); ok {
		parsed, err := parseList(raw)
		if err != nil {
			c.loadErrors = append(c.loadErrors, fmt.Errorf("invalid AVAILABILITY_SHARDS: %w", err))
		} else {
			c.Shards = parsed
		}
	}

	if raw, ok := lookupEnv("AVAILABILITY_PARTITION_MAP"); ok {
		parsed, err := parsePartitionMap(raw)
		if err != nil {
			c.loadErrors = append(c.loadErrors, fmt.Errorf("invalid AVAILABILITY_PARTITION_MAP: %w", err))
		} else {
			c.PartitionMap = parsed
		}
	}

	c.ShardDSNs = make(map[string]string, len(c.Shards))
	for _, shard := range c.Shards {
		if raw, ok := lookupEnv(shardDSNEnvName(shard)); ok {
			c.ShardDSNs[shard] = strings.TrimSpace(raw)
		}
	}
}

func parseBool(raw string) (bool, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "true", "t", "1", "yes", "y", "on", "enabled":
		return true, nil
	case "false", "f", "0", "no", "n", "off", "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("expected boolean value, got %q", raw)
	}
}

func parsePositiveInt(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, errors.New("value cannot be empty")
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, errors.New("value must be greater than zero")
	}

	return parsed, nil
}

func parseList(raw string) ([]string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("value cannot be empty")
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for i, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			return nil, fmt.Errorf("empty item at position %d", i)
		}
		items = append(items, item)
	}

	return items, nil
}

func parsePartitionMap(raw string) (map[int]string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("value cannot be empty")
	}

	value = strings.ReplaceAll(value, ";", ",")
	entries := strings.Split(value, ",")
	partitionMap := make(map[int]string, len(entries))
	for _, entry := range entries {
		key, shard, ok := splitMapEntry(entry)
		if !ok {
			return nil, fmt.Errorf("entry %q must use partition:shard or partition=shard", entry)
		}

		partition, err := parseNonNegativeInt(key)
		if err != nil {
			return nil, fmt.Errorf("invalid partition %q: %w", key, err)
		}
		if shard == "" {
			return nil, fmt.Errorf("partition %d has empty shard", partition)
		}
		if _, exists := partitionMap[partition]; exists {
			return nil, fmt.Errorf("partition %d is configured more than once", partition)
		}

		partitionMap[partition] = shard
	}

	return partitionMap, nil
}

func splitMapEntry(entry string) (key string, shard string, ok bool) {
	entry = strings.TrimSpace(entry)
	for _, sep := range []string{":", "="} {
		parts := strings.SplitN(entry, sep, 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
		}
	}

	return "", "", false
}

func parseNonNegativeInt(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, errors.New("value cannot be empty")
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if parsed < 0 {
		return 0, errors.New("value must be greater than or equal to zero")
	}

	return parsed, nil
}

func shardDSNEnvName(shard string) string {
	var builder strings.Builder
	builder.WriteString("AVAILABILITY_")
	for _, r := range shard {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(unicode.ToUpper(r))
			continue
		}
		builder.WriteRune('_')
	}
	builder.WriteString("_DSN")

	return builder.String()
}

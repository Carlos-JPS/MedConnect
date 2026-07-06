package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Carlos-JPS/medconnect/availability-service/modules/config"
	"github.com/Carlos-JPS/medconnect/availability-service/modules/handler"
	"github.com/Carlos-JPS/medconnect/availability-service/modules/observability"
	"github.com/Carlos-JPS/medconnect/availability-service/modules/repository"
	"github.com/Carlos-JPS/medconnect/availability-service/modules/service"
	"github.com/Carlos-JPS/medconnect/availability-service/modules/sharding"
	pb "github.com/Carlos-JPS/medconnect/availability-service/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	dbConnectMaxAttempts = 5
	dbConnectRetryDelay  = 3 * time.Second
)

type slotIDLister interface {
	ListSlotIDs(ctx context.Context) ([]string, error)
}

func main() {
	cfg := config.Load()

	repo, err := buildRepository(context.Background(), cfg)
	if err != nil {
		log.Fatalf("no se pudo inicializar el repositorio de disponibilidad: %v", err)
	}

	svc := service.NewAvailabilityService(repo)
	h := handler.NewGRPCHandler(svc)

	addr := fmt.Sprintf(":%s", cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("error abriendo puerto %s: %v", addr, err)
	}

	observability.StartMetricsServer("availability-service")
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(observability.UnaryServerInterceptor("availability-service")))
	pb.RegisterAvailabilityServiceServer(grpcServer, h)
	reflection.Register(grpcServer)

	log.Printf("availability-service escuchando en %s (gRPC)", addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("error en servidor gRPC: %v", err)
	}
}

func buildRepository(ctx context.Context, cfg *config.Config) (repository.AvailabilityRepository, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config no puede ser nil")
	}

	if !cfg.ShardingEnabled {
		log.Println("availability-service iniciando en modo single DB")
		repo, err := connectWithRetry(cfg.DSN(), dbConnectMaxAttempts, dbConnectRetryDelay)
		if err != nil {
			return nil, fmt.Errorf("conectar a PostgreSQL single DB: %w", err)
		}
		log.Println("conexión a PostgreSQL establecida en modo single DB")
		return repo, nil
	}

	if err := cfg.ValidateShardingConfig(); err != nil {
		return nil, fmt.Errorf("configuración sharded inválida: %w", err)
	}

	router, err := sharding.NewRouter(cfg.PartitionCount, cfg.PartitionMap)
	if err != nil {
		return nil, fmt.Errorf("crear router de sharding: %w", err)
	}

	shardNames := cfg.Shards
	log.Printf(
		"availability-service iniciando en modo sharded: particiones=%d shards_físicos=%d names=%v shards_enrutados=%v",
		cfg.PartitionCount,
		len(shardNames),
		shardNames,
		router.AllShards(),
	)

	shardRepos := make(map[string]repository.AvailabilityRepository, len(shardNames))
	slotListers := make(map[string]slotIDLister, len(shardNames))
	for _, shardName := range shardNames {
		dsn := cfg.ShardDSNs[shardName]
		if dsn == "" {
			return nil, fmt.Errorf("shard %q no tiene DSN configurado", shardName)
		}

		repo, err := connectWithRetry(dsn, dbConnectMaxAttempts, dbConnectRetryDelay)
		if err != nil {
			return nil, fmt.Errorf("conectar shard %q: %w", shardName, err)
		}

		shardRepos[shardName] = repo
		slotListers[shardName] = repo
		log.Printf("conexión a PostgreSQL establecida para shard %q", shardName)
	}

	slotDirectory, err := buildSlotDirectory(ctx, slotListers)
	if err != nil {
		return nil, fmt.Errorf("construir directorio slot_id->shard: %w", err)
	}
	log.Printf("directorio de slots sharded construido con %d entradas", len(slotDirectory))

	repo, err := repository.NewShardedRepository(router, shardRepos, slotDirectory)
	if err != nil {
		return nil, fmt.Errorf("crear ShardedRepository: %w", err)
	}

	log.Printf("ShardedRepository listo: shards=%d entradas_directorio=%d", len(shardRepos), len(slotDirectory))
	return repo, nil
}

func buildSlotDirectory(ctx context.Context, shardListers map[string]slotIDLister) (map[string]string, error) {
	slotDirectory := make(map[string]string)
	for shardName, lister := range shardListers {
		if shardName == "" {
			return nil, fmt.Errorf("nombre de shard vacío")
		}
		if lister == nil {
			return nil, fmt.Errorf("slot lister nil para shard %q", shardName)
		}

		slotIDs, err := lister.ListSlotIDs(ctx)
		if err != nil {
			return nil, fmt.Errorf("listar slots en shard %q: %w", shardName, err)
		}

		for _, slotID := range slotIDs {
			if slotID == "" {
				return nil, fmt.Errorf("shard %q retornó slot_id vacío", shardName)
			}
			if existingShard, exists := slotDirectory[slotID]; exists {
				return nil, fmt.Errorf("slot_id %q existe en más de un shard (%q y %q)", slotID, existingShard, shardName)
			}
			slotDirectory[slotID] = shardName
		}
		log.Printf("shard %q aportó %d slots al directorio", shardName, len(slotIDs))
	}

	return slotDirectory, nil
}

func connectWithRetry(dsn string, maxAttempts int, delay time.Duration) (*repository.PostgresRepository, error) {
	var repo *repository.PostgresRepository
	var err error

	for i := 1; i <= maxAttempts; i++ {
		repo, err = repository.NewPostgresRepository(dsn)
		if err == nil {
			return repo, nil
		}
		log.Printf("intento %d/%d fallido al conectar a postgres: %v", i, maxAttempts, err)
		if i < maxAttempts {
			time.Sleep(delay)
		}
	}
	return nil, fmt.Errorf("todos los intentos de conexión fallaron: %w", err)
}

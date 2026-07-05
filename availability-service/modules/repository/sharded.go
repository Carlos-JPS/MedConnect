package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Carlos-JPS/medconnect/availability-service/modules/sharding"
)

var ErrSlotShardNotFound = errors.New("slot has no shard directory entry")

// ShardedRepository routes availability operations to the repository that owns
// each shard. Reads by doctor_id use the router; reads by specialty fan out to
// all router shards; mutations by slot_id use the slot directory.
type ShardedRepository struct {
	router        *sharding.Router
	shards        map[string]AvailabilityRepository
	slotDirectory map[string]string
}

var _ AvailabilityRepository = (*ShardedRepository)(nil)

func NewShardedRepository(router *sharding.Router, shards map[string]AvailabilityRepository, slotDirectory map[string]string) (*ShardedRepository, error) {
	if router == nil {
		return nil, fmt.Errorf("sharding router cannot be nil")
	}
	if len(shards) == 0 {
		return nil, fmt.Errorf("at least one shard repository is required")
	}

	shardCopies := make(map[string]AvailabilityRepository, len(shards))
	for name, repo := range shards {
		if name == "" {
			return nil, fmt.Errorf("shard name cannot be empty")
		}
		if repo == nil {
			return nil, fmt.Errorf("repository for shard %q cannot be nil", name)
		}
		shardCopies[name] = repo
	}

	for _, shardName := range router.AllShards() {
		if _, ok := shardCopies[shardName]; !ok {
			return nil, fmt.Errorf("router requires shard %q but no repository was provided", shardName)
		}
	}

	directoryCopy := make(map[string]string, len(slotDirectory))
	for slotID, shardName := range slotDirectory {
		if _, ok := shardCopies[shardName]; !ok {
			return nil, fmt.Errorf("slot directory maps slot %q to unknown shard %q", slotID, shardName)
		}
		directoryCopy[slotID] = shardName
	}

	return &ShardedRepository{
		router:        router,
		shards:        shardCopies,
		slotDirectory: directoryCopy,
	}, nil
}

func (r *ShardedRepository) GetDoctorAgenda(ctx context.Context, doctorID string, startDate, endDate time.Time) ([]*Slot, error) {
	route, err := r.router.ShardForDoctor(doctorID)
	if err != nil {
		return nil, fmt.Errorf("route doctor agenda: %w", err)
	}

	repo, err := r.repositoryForShard(route.Shard)
	if err != nil {
		return nil, err
	}

	return repo.GetDoctorAgenda(ctx, doctorID, startDate, endDate)
}

func (r *ShardedRepository) GetAvailableSlots(ctx context.Context, specialty string, startDate, endDate time.Time) ([]*Slot, error) {
	shardNames := r.router.AllShards()
	results := make(chan shardSlotsResult, len(shardNames))

	var wg sync.WaitGroup
	for _, shardName := range shardNames {
		repo, err := r.repositoryForShard(shardName)
		if err != nil {
			return nil, err
		}

		wg.Add(1)
		go func(shardName string, repo AvailabilityRepository) {
			defer wg.Done()

			slots, err := repo.GetAvailableSlots(ctx, specialty, startDate, endDate)
			results <- shardSlotsResult{shard: shardName, slots: slots, err: err}
		}(shardName, repo)
	}

	wg.Wait()
	close(results)

	var allSlots []*Slot
	for result := range results {
		if result.err != nil {
			return nil, fmt.Errorf("query available slots on shard %q: %w", result.shard, result.err)
		}
		allSlots = append(allSlots, result.slots...)
	}

	sort.SliceStable(allSlots, func(i, j int) bool {
		return allSlots[i].StartTime.Before(allSlots[j].StartTime)
	})

	return allSlots, nil
}

func (r *ShardedRepository) HoldSlot(ctx context.Context, slotID string, bookingID string, heldUntil time.Time) (*Slot, error) {
	repo, err := r.repositoryForSlot(slotID)
	if err != nil {
		return nil, err
	}

	return repo.HoldSlot(ctx, slotID, bookingID, heldUntil)
}

func (r *ShardedRepository) ConfirmSlotBooking(ctx context.Context, slotID string, bookingID string) (*Slot, error) {
	repo, err := r.repositoryForSlot(slotID)
	if err != nil {
		return nil, err
	}

	return repo.ConfirmSlotBooking(ctx, slotID, bookingID)
}

func (r *ShardedRepository) ReleaseHeldSlot(ctx context.Context, slotID string, bookingID string) (*Slot, error) {
	repo, err := r.repositoryForSlot(slotID)
	if err != nil {
		return nil, err
	}

	return repo.ReleaseHeldSlot(ctx, slotID, bookingID)
}

func (r *ShardedRepository) repositoryForSlot(slotID string) (AvailabilityRepository, error) {
	shardName, ok := r.slotDirectory[slotID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrSlotShardNotFound, slotID)
	}

	repo, err := r.repositoryForShard(shardName)
	if err != nil {
		return nil, fmt.Errorf("slot %q is mapped to invalid shard: %w", slotID, err)
	}

	return repo, nil
}

func (r *ShardedRepository) repositoryForShard(shardName string) (AvailabilityRepository, error) {
	repo, ok := r.shards[shardName]
	if !ok {
		return nil, fmt.Errorf("shard %q has no repository configured", shardName)
	}
	return repo, nil
}

type shardSlotsResult struct {
	shard string
	slots []*Slot
	err   error
}

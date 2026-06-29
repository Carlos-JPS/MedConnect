package repository

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Carlos-JPS/medconnect/availability-service/modules/sharding"
)

func TestShardedRepositoryGetDoctorAgendaRoutesSingleShardByDoctor(t *testing.T) {
	ctx := context.Background()
	router := mustRouter(t, 1, map[int]string{0: "shard-a"})
	start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	shardA := &fakeAvailabilityRepository{
		agenda: []*Slot{{ID: "agenda-1", StartTime: start}},
	}
	shardB := &fakeAvailabilityRepository{
		agendaErr: errors.New("should not be called"),
	}
	repo := mustShardedRepository(t, router, map[string]AvailabilityRepository{
		"shard-a": shardA,
		"shard-b": shardB,
	}, nil)

	slots, err := repo.GetDoctorAgenda(ctx, "doctor-123", start, end)
	if err != nil {
		t.Fatalf("GetDoctorAgenda returned error: %v", err)
	}

	if got := shardA.callCount("GetDoctorAgenda"); got != 1 {
		t.Fatalf("expected shard-a to be called once, got %d", got)
	}
	if got := shardB.callCount("GetDoctorAgenda"); got != 0 {
		t.Fatalf("expected shard-b not to be called, got %d", got)
	}
	if got, want := slotIDs(slots), []string{"agenda-1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected slots %v, got %v", want, got)
	}
}

func TestShardedRepositoryGetAvailableSlotsScatterGatherSorted(t *testing.T) {
	ctx := context.Background()
	router := mustRouter(t, 2, map[int]string{0: "shard-a", 1: "shard-b"})
	base := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	shardA := &fakeAvailabilityRepository{
		available: []*Slot{{ID: "slot-3", StartTime: base.Add(2 * time.Hour)}},
	}
	shardB := &fakeAvailabilityRepository{
		available: []*Slot{
			{ID: "slot-2", StartTime: base.Add(time.Hour)},
			{ID: "slot-1", StartTime: base},
		},
	}
	repo := mustShardedRepository(t, router, map[string]AvailabilityRepository{
		"shard-a": shardA,
		"shard-b": shardB,
	}, nil)

	slots, err := repo.GetAvailableSlots(ctx, "Cardiología", base, base.Add(3*time.Hour))
	if err != nil {
		t.Fatalf("GetAvailableSlots returned error: %v", err)
	}

	if got := shardA.callCount("GetAvailableSlots"); got != 1 {
		t.Fatalf("expected shard-a to be queried once, got %d", got)
	}
	if got := shardB.callCount("GetAvailableSlots"); got != 1 {
		t.Fatalf("expected shard-b to be queried once, got %d", got)
	}
	if got, want := slotIDs(slots), []string{"slot-1", "slot-2", "slot-3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected sorted slots %v, got %v", want, got)
	}
}

func TestShardedRepositoryGetAvailableSlotsFailsWhenAnyShardFails(t *testing.T) {
	ctx := context.Background()
	router := mustRouter(t, 2, map[int]string{0: "shard-a", 1: "shard-b"})
	base := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	shardA := &fakeAvailabilityRepository{
		available: []*Slot{{ID: "slot-ok", StartTime: base}},
	}
	shardB := &fakeAvailabilityRepository{
		availableErr: errors.New("database unavailable"),
	}
	repo := mustShardedRepository(t, router, map[string]AvailabilityRepository{
		"shard-a": shardA,
		"shard-b": shardB,
	}, nil)

	slots, err := repo.GetAvailableSlots(ctx, "Cardiología", base, base.Add(time.Hour))
	if err == nil {
		t.Fatal("expected error when one shard fails")
	}
	if slots != nil {
		t.Fatalf("expected no partial result on shard failure, got %v", slotIDs(slots))
	}
	if !strings.Contains(err.Error(), "shard-b") {
		t.Fatalf("expected error to mention failing shard, got %v", err)
	}
}

func TestShardedRepositoryMutationsRouteBySlotDirectory(t *testing.T) {
	ctx := context.Background()
	router := mustRouter(t, 2, map[int]string{0: "shard-a", 1: "shard-b"})
	shardA := &fakeAvailabilityRepository{}
	shardB := &fakeAvailabilityRepository{}
	repo := mustShardedRepository(t, router, map[string]AvailabilityRepository{
		"shard-a": shardA,
		"shard-b": shardB,
	}, map[string]string{"slot-b": "shard-b"})

	heldUntil := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		method string
		call   func() (*Slot, error)
	}{
		{
			name:   "hold",
			method: "HoldSlot",
			call: func() (*Slot, error) {
				return repo.HoldSlot(ctx, "slot-b", "booking-1", heldUntil)
			},
		},
		{
			name:   "confirm",
			method: "ConfirmSlotBooking",
			call: func() (*Slot, error) {
				return repo.ConfirmSlotBooking(ctx, "slot-b", "booking-1")
			},
		},
		{
			name:   "release",
			method: "ReleaseHeldSlot",
			call: func() (*Slot, error) {
				return repo.ReleaseHeldSlot(ctx, "slot-b", "booking-1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slot, err := tt.call()
			if err != nil {
				t.Fatalf("%s returned error: %v", tt.method, err)
			}
			if slot.ID != "slot-b" {
				t.Fatalf("expected slot-b, got %q", slot.ID)
			}
			if got := shardA.callCount(tt.method); got != 0 {
				t.Fatalf("expected shard-a not to handle %s, got %d calls", tt.method, got)
			}
			if got := shardB.callCount(tt.method); got != 1 {
				t.Fatalf("expected shard-b to handle %s once, got %d", tt.method, got)
			}
			shardB.resetCalls()
		})
	}
}

func TestShardedRepositoryMutationsFailWhenSlotIsMissingFromDirectory(t *testing.T) {
	ctx := context.Background()
	router := mustRouter(t, 1, map[int]string{0: "shard-a"})
	repo := mustShardedRepository(t, router, map[string]AvailabilityRepository{
		"shard-a": &fakeAvailabilityRepository{},
	}, nil)

	heldUntil := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		call func() (*Slot, error)
	}{
		{
			name: "hold",
			call: func() (*Slot, error) {
				return repo.HoldSlot(ctx, "missing-slot", "booking-1", heldUntil)
			},
		},
		{
			name: "confirm",
			call: func() (*Slot, error) {
				return repo.ConfirmSlotBooking(ctx, "missing-slot", "booking-1")
			},
		},
		{
			name: "release",
			call: func() (*Slot, error) {
				return repo.ReleaseHeldSlot(ctx, "missing-slot", "booking-1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.call(); !errors.Is(err, ErrSlotShardNotFound) {
				t.Fatalf("expected ErrSlotShardNotFound, got %v", err)
			}
		})
	}
}

func TestNewShardedRepositoryValidations(t *testing.T) {
	validRouter := mustRouter(t, 1, map[int]string{0: "shard-a"})
	multiShardRouter := mustRouter(t, 2, map[int]string{0: "shard-a", 1: "shard-b"})
	validShards := map[string]AvailabilityRepository{"shard-a": &fakeAvailabilityRepository{}}

	tests := []struct {
		name          string
		router        *sharding.Router
		shards        map[string]AvailabilityRepository
		slotDirectory map[string]string
		wantErr       string
	}{
		{
			name:    "nil router",
			router:  nil,
			shards:  validShards,
			wantErr: "router",
		},
		{
			name:    "empty shards",
			router:  validRouter,
			shards:  map[string]AvailabilityRepository{},
			wantErr: "at least one shard",
		},
		{
			name:    "missing shard required by router",
			router:  multiShardRouter,
			shards:  map[string]AvailabilityRepository{"shard-a": &fakeAvailabilityRepository{}},
			wantErr: "shard-b",
		},
		{
			name:          "directory points to unknown shard",
			router:        validRouter,
			shards:        validShards,
			slotDirectory: map[string]string{"slot-1": "shard-missing"},
			wantErr:       "unknown shard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewShardedRepository(tt.router, tt.shards, tt.slotDirectory)
			if err == nil {
				t.Fatal("expected constructor error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestNewShardedRepositoryCopiesInputMaps(t *testing.T) {
	ctx := context.Background()
	router := mustRouter(t, 1, map[int]string{0: "shard-a"})
	shardA := &fakeAvailabilityRepository{}
	shardB := &fakeAvailabilityRepository{}
	shards := map[string]AvailabilityRepository{"shard-a": shardA}
	slotDirectory := map[string]string{"slot-1": "shard-a"}
	repo := mustShardedRepository(t, router, shards, slotDirectory)

	shards["shard-a"] = shardB
	shards["shard-b"] = shardB
	slotDirectory["slot-1"] = "shard-b"

	if _, err := repo.HoldSlot(ctx, "slot-1", "booking-1", time.Now().UTC()); err != nil {
		t.Fatalf("HoldSlot returned error: %v", err)
	}

	if got := shardA.callCount("HoldSlot"); got != 1 {
		t.Fatalf("expected original shard-a repository to be called once, got %d", got)
	}
	if got := shardB.callCount("HoldSlot"); got != 0 {
		t.Fatalf("expected replacement shard-b repository not to be called, got %d", got)
	}
}

type fakeAvailabilityRepository struct {
	mu sync.Mutex

	available    []*Slot
	availableErr error
	agenda       []*Slot
	agendaErr    error
	calls        map[string]int
}

func (f *fakeAvailabilityRepository) GetAvailableSlots(ctx context.Context, specialty string, startDate, endDate time.Time) ([]*Slot, error) {
	f.recordCall("GetAvailableSlots")
	if f.availableErr != nil {
		return nil, f.availableErr
	}
	return f.available, nil
}

func (f *fakeAvailabilityRepository) HoldSlot(ctx context.Context, slotID string, bookingID string, heldUntil time.Time) (*Slot, error) {
	f.recordCall("HoldSlot")
	return &Slot{ID: slotID, Status: StatusHeld}, nil
}

func (f *fakeAvailabilityRepository) ConfirmSlotBooking(ctx context.Context, slotID string, bookingID string) (*Slot, error) {
	f.recordCall("ConfirmSlotBooking")
	return &Slot{ID: slotID, Status: StatusBooked}, nil
}

func (f *fakeAvailabilityRepository) ReleaseHeldSlot(ctx context.Context, slotID string, bookingID string) (*Slot, error) {
	f.recordCall("ReleaseHeldSlot")
	return &Slot{ID: slotID, Status: StatusAvailable}, nil
}

func (f *fakeAvailabilityRepository) GetDoctorAgenda(ctx context.Context, doctorID string, startDate, endDate time.Time) ([]*Slot, error) {
	f.recordCall("GetDoctorAgenda")
	if f.agendaErr != nil {
		return nil, f.agendaErr
	}
	return f.agenda, nil
}

func (f *fakeAvailabilityRepository) recordCall(method string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.calls == nil {
		f.calls = make(map[string]int)
	}
	f.calls[method]++
}

func (f *fakeAvailabilityRepository) callCount(method string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[method]
}

func (f *fakeAvailabilityRepository) resetCalls() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = make(map[string]int)
}

func mustRouter(t *testing.T, partitionCount int, partitionMap map[int]string) *sharding.Router {
	t.Helper()

	router, err := sharding.NewRouter(partitionCount, partitionMap)
	if err != nil {
		t.Fatalf("NewRouter returned error: %v", err)
	}
	return router
}

func mustShardedRepository(t *testing.T, router *sharding.Router, shards map[string]AvailabilityRepository, slotDirectory map[string]string) *ShardedRepository {
	t.Helper()

	repo, err := NewShardedRepository(router, shards, slotDirectory)
	if err != nil {
		t.Fatalf("NewShardedRepository returned error: %v", err)
	}
	return repo
}

func slotIDs(slots []*Slot) []string {
	ids := make([]string, 0, len(slots))
	for _, slot := range slots {
		ids = append(ids, slot.ID)
	}
	return ids
}

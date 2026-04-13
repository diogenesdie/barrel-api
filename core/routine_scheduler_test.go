package core_test

import (
	"barrel-api/core"
	"barrel-api/model"
	"barrel-api/repository"
	"testing"
	"time"
)

func newTestScheduler(repo repository.RoutineRepositoryInterface, executor core.RoutineRunnerIface) *core.RoutineScheduler {
	return core.NewRoutineScheduler(repo, executor)
}

// mockRoutineRepo for scheduler tests
type schedMockRoutineRepo struct {
	routines map[uint64]*model.Routine
	nextID   uint64
}

func newSchedMockRoutineRepo() *schedMockRoutineRepo {
	return &schedMockRoutineRepo{routines: make(map[uint64]*model.Routine), nextID: 1}
}

func (m *schedMockRoutineRepo) CreateRoutine(r *model.Routine) (uint64, error) {
	id := m.nextID
	m.nextID++
	r.ID = id
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()
	m.routines[id] = r
	return id, nil
}
func (m *schedMockRoutineRepo) GetRoutineByID(id uint64) (*model.Routine, error) {
	r, ok := m.routines[id]
	if !ok {
		return nil, repository.ErrRoutineNotFound
	}
	return r, nil
}
func (m *schedMockRoutineRepo) GetRoutinesByUser(userID uint64) ([]model.Routine, error) {
	return nil, nil
}
func (m *schedMockRoutineRepo) GetEnabledRoutinesByTriggerType(t string) ([]model.Routine, error) {
	result := []model.Routine{}
	for _, r := range m.routines {
		if r.Enabled && r.Trigger.Type == t {
			result = append(result, *r)
		}
	}
	return result, nil
}
func (m *schedMockRoutineRepo) UpdateRoutine(r *model.Routine) error {
	if _, ok := m.routines[r.ID]; !ok {
		return repository.ErrRoutineNotFound
	}
	m.routines[r.ID] = r
	return nil
}
func (m *schedMockRoutineRepo) DeleteRoutine(id uint64, userID uint64) error {
	delete(m.routines, id)
	return nil
}

// mockExecutor for scheduler tests — counts executions
type schedMockExecutor struct {
	execCount int
}

func (m *schedMockExecutor) ExecuteRoutine(r *model.Routine) error {
	m.execCount++
	return nil
}

func cronPtr(s string) *string { return &s }

func TestScheduler_AddAndRemoveJob(t *testing.T) {
	repo := newSchedMockRoutineRepo()
	executor := &schedMockExecutor{}
	scheduler := newTestScheduler(repo, executor)

	cronExpr := "* * * * *" // every minute
	if err := scheduler.AddRoutine(1, 1, cronExpr, "Teste"); err != nil {
		t.Fatalf("expected no error adding job, got: %v", err)
	}

	if len(scheduler.JobIDs()) != 1 {
		t.Errorf("expected 1 job registered, got %d", len(scheduler.JobIDs()))
	}

	scheduler.RemoveRoutine(1)
	if len(scheduler.JobIDs()) != 0 {
		t.Errorf("expected 0 jobs after removal, got %d", len(scheduler.JobIDs()))
	}
}

func TestScheduler_InvalidCronExpr(t *testing.T) {
	repo := newSchedMockRoutineRepo()
	executor := &schedMockExecutor{}
	scheduler := newTestScheduler(repo, executor)

	err := scheduler.AddRoutine(1, 1, "not-a-cron", "Inválida")
	if err == nil {
		t.Fatal("expected error for invalid cron expression, got nil")
	}
}

func TestScheduler_ReplacesExistingJob(t *testing.T) {
	repo := newSchedMockRoutineRepo()
	executor := &schedMockExecutor{}
	scheduler := newTestScheduler(repo, executor)

	scheduler.AddRoutine(1, 1, "* * * * *", "Original")
	scheduler.AddRoutine(1, 1, "0 8 * * *", "Atualizada")

	if len(scheduler.JobIDs()) != 1 {
		t.Errorf("expected 1 job after replacement, got %d", len(scheduler.JobIDs()))
	}
}

func TestScheduler_LoadsEnabledRoutinesOnStart(t *testing.T) {
	repo := newSchedMockRoutineRepo()
	cron1 := "0 8 * * *"
	cron2 := "0 22 * * *"
	repo.CreateRoutine(&model.Routine{
		UserID: 1, Name: "Manhã", Enabled: true,
		Trigger: model.RoutineTrigger{Type: "schedule", Cron: &cron1},
	})
	repo.CreateRoutine(&model.Routine{
		UserID: 1, Name: "Noite", Enabled: true,
		Trigger: model.RoutineTrigger{Type: "schedule", Cron: &cron2},
	})
	repo.CreateRoutine(&model.Routine{
		UserID: 1, Name: "Disabled", Enabled: false,
		Trigger: model.RoutineTrigger{Type: "schedule", Cron: &cron1},
	})

	executor := &schedMockExecutor{}
	scheduler := newTestScheduler(repo, executor)

	if err := scheduler.Start(); err != nil {
		t.Fatalf("expected no error on start, got: %v", err)
	}
	defer scheduler.Stop()

	if len(scheduler.JobIDs()) != 2 {
		t.Errorf("expected 2 jobs (enabled only), got %d", len(scheduler.JobIDs()))
	}
}

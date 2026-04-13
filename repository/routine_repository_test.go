package repository_test

import (
	"barrel-api/model"
	"barrel-api/repository"
	"testing"
	"time"
)

// mockRoutineRepository implements RoutineRepositoryInterface for unit testing.
type mockRoutineRepository struct {
	routines map[uint64]*model.Routine
	nextID   uint64
}

func newMockRoutineRepository() *mockRoutineRepository {
	return &mockRoutineRepository{
		routines: make(map[uint64]*model.Routine),
		nextID:   1,
	}
}

func (m *mockRoutineRepository) CreateRoutine(r *model.Routine) (uint64, error) {
	id := m.nextID
	m.nextID++
	r.ID = id
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()
	m.routines[id] = r
	return id, nil
}

func (m *mockRoutineRepository) GetRoutineByID(id uint64) (*model.Routine, error) {
	r, ok := m.routines[id]
	if !ok {
		return nil, repository.ErrRoutineNotFound
	}
	return r, nil
}

func (m *mockRoutineRepository) GetRoutinesByUser(userID uint64) ([]model.Routine, error) {
	result := []model.Routine{}
	for _, r := range m.routines {
		if r.UserID == userID {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *mockRoutineRepository) GetEnabledRoutinesByTriggerType(triggerType string) ([]model.Routine, error) {
	result := []model.Routine{}
	for _, r := range m.routines {
		if r.Enabled && r.Trigger.Type == triggerType {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *mockRoutineRepository) UpdateRoutine(r *model.Routine) error {
	if _, ok := m.routines[r.ID]; !ok {
		return repository.ErrRoutineNotFound
	}
	r.UpdatedAt = time.Now()
	m.routines[r.ID] = r
	return nil
}

func (m *mockRoutineRepository) DeleteRoutine(id uint64, userID uint64) error {
	r, ok := m.routines[id]
	if !ok {
		return repository.ErrRoutineNotFound
	}
	if r.UserID != userID {
		return repository.ErrRoutineNotFound
	}
	delete(m.routines, id)
	return nil
}

// Tests

func TestCreateRoutine(t *testing.T) {
	repo := newMockRoutineRepository()
	routine := &model.Routine{
		UserID:  1,
		Name:    "Chegou em casa",
		Enabled: true,
		Trigger: model.RoutineTrigger{
			Type:          "device",
			DeviceID:      uintPtr(42),
			ExpectedState: map[string]string{"power": "on"},
		},
		Actions: []model.RoutineAction{
			{Type: "device", DeviceID: uintPtr(10), Command: strPtr("on"), SortOrder: 1},
			{Type: "scene", SceneID: uintPtr(5), SortOrder: 2},
		},
	}

	id, err := repo.CreateRoutine(routine)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestGetRoutineByID_Found(t *testing.T) {
	repo := newMockRoutineRepository()
	routine := &model.Routine{
		UserID:  1,
		Name:    "Rotina Teste",
		Enabled: true,
		Trigger: model.RoutineTrigger{Type: "schedule", Cron: strPtr("0 22 * * *")},
	}
	id, _ := repo.CreateRoutine(routine)

	got, err := repo.GetRoutineByID(id)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got.Name != "Rotina Teste" {
		t.Errorf("expected name 'Rotina Teste', got '%s'", got.Name)
	}
}

func TestGetRoutineByID_NotFound(t *testing.T) {
	repo := newMockRoutineRepository()
	_, err := repo.GetRoutineByID(999)
	if err != repository.ErrRoutineNotFound {
		t.Fatalf("expected ErrRoutineNotFound, got: %v", err)
	}
}

func TestGetRoutinesByUser(t *testing.T) {
	repo := newMockRoutineRepository()
	repo.CreateRoutine(&model.Routine{UserID: 1, Name: "Rotina A", Trigger: model.RoutineTrigger{Type: "schedule"}})
	repo.CreateRoutine(&model.Routine{UserID: 1, Name: "Rotina B", Trigger: model.RoutineTrigger{Type: "device"}})
	repo.CreateRoutine(&model.Routine{UserID: 2, Name: "Rotina C", Trigger: model.RoutineTrigger{Type: "schedule"}})

	routines, err := repo.GetRoutinesByUser(1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(routines) != 2 {
		t.Errorf("expected 2 routines for user 1, got %d", len(routines))
	}
}

func TestGetEnabledRoutinesByTriggerType(t *testing.T) {
	repo := newMockRoutineRepository()
	repo.CreateRoutine(&model.Routine{UserID: 1, Enabled: true, Name: "A", Trigger: model.RoutineTrigger{Type: "device"}})
	repo.CreateRoutine(&model.Routine{UserID: 1, Enabled: false, Name: "B", Trigger: model.RoutineTrigger{Type: "device"}})
	repo.CreateRoutine(&model.Routine{UserID: 1, Enabled: true, Name: "C", Trigger: model.RoutineTrigger{Type: "schedule"}})

	deviceRoutines, err := repo.GetEnabledRoutinesByTriggerType("device")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(deviceRoutines) != 1 {
		t.Errorf("expected 1 enabled device routine, got %d", len(deviceRoutines))
	}
}

func TestUpdateRoutine(t *testing.T) {
	repo := newMockRoutineRepository()
	routine := &model.Routine{UserID: 1, Name: "Original", Trigger: model.RoutineTrigger{Type: "schedule"}}
	id, _ := repo.CreateRoutine(routine)

	routine.ID = id
	routine.Name = "Atualizada"
	if err := repo.UpdateRoutine(routine); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := repo.GetRoutineByID(id)
	if got.Name != "Atualizada" {
		t.Errorf("expected 'Atualizada', got '%s'", got.Name)
	}
}

func TestUpdateRoutine_NotFound(t *testing.T) {
	repo := newMockRoutineRepository()
	err := repo.UpdateRoutine(&model.Routine{ID: 999, UserID: 1, Trigger: model.RoutineTrigger{Type: "schedule"}})
	if err != repository.ErrRoutineNotFound {
		t.Fatalf("expected ErrRoutineNotFound, got: %v", err)
	}
}

func TestDeleteRoutine(t *testing.T) {
	repo := newMockRoutineRepository()
	routine := &model.Routine{UserID: 1, Name: "Para deletar", Trigger: model.RoutineTrigger{Type: "schedule"}}
	id, _ := repo.CreateRoutine(routine)

	if err := repo.DeleteRoutine(id, 1); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if _, err := repo.GetRoutineByID(id); err != repository.ErrRoutineNotFound {
		t.Fatal("expected routine to be deleted")
	}
}

func TestDeleteRoutine_NotFound(t *testing.T) {
	repo := newMockRoutineRepository()
	err := repo.DeleteRoutine(999, 1)
	if err != repository.ErrRoutineNotFound {
		t.Fatalf("expected ErrRoutineNotFound, got: %v", err)
	}
}

func uintPtr(v uint64) *uint64 { return &v }

package controller_test

import (
	"barrel-api/controller"
	"barrel-api/model"
	"barrel-api/repository"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// --- mock routine repository ---

type mockRoutineRepo struct {
	routines map[uint64]*model.Routine
	nextID   uint64
}

func newMockRoutineRepo() *mockRoutineRepo {
	return &mockRoutineRepo{routines: make(map[uint64]*model.Routine), nextID: 1}
}

func (m *mockRoutineRepo) CreateRoutine(r *model.Routine) (uint64, error) {
	id := m.nextID
	m.nextID++
	r.ID = id
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()
	m.routines[id] = r
	return id, nil
}

func (m *mockRoutineRepo) GetRoutineByID(id uint64) (*model.Routine, error) {
	r, ok := m.routines[id]
	if !ok {
		return nil, repository.ErrRoutineNotFound
	}
	return r, nil
}

func (m *mockRoutineRepo) GetRoutinesByUser(userID uint64) ([]model.Routine, error) {
	result := []model.Routine{}
	for _, r := range m.routines {
		if r.UserID == userID {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *mockRoutineRepo) GetEnabledRoutinesByTriggerType(triggerType string) ([]model.Routine, error) {
	result := []model.Routine{}
	for _, r := range m.routines {
		if r.Enabled && r.Trigger.Type == triggerType {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *mockRoutineRepo) UpdateRoutine(r *model.Routine) error {
	if _, ok := m.routines[r.ID]; !ok {
		return repository.ErrRoutineNotFound
	}
	m.routines[r.ID] = r
	return nil
}

func (m *mockRoutineRepo) DeleteRoutine(id uint64, userID uint64) error {
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

// --- mock routine executor ---

type mockRoutineExecutor struct{}

func (m *mockRoutineExecutor) ExecuteRoutine(routine *model.Routine) error {
	return nil
}

// --- helpers ---

func makeRoutineRequest(method, path string, body interface{}, userID uint64) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("user_id", fmt.Sprintf("%d", userID))
	return req
}

func makeRoutineRouter(routineRepo repository.RoutineRepositoryInterface, executor controller.RoutineExecutorInterface) *mux.Router {
	r := mux.NewRouter()
	c := controller.NewRoutineController(routineRepo, executor)
	r.HandleFunc("/api/v1/routines", c.ListRoutinesHandler).Methods("GET")
	r.HandleFunc("/api/v1/routines", c.CreateRoutineHandler).Methods("POST")
	r.HandleFunc("/api/v1/routines/{id}", c.GetRoutineHandler).Methods("GET")
	r.HandleFunc("/api/v1/routines/{id}", c.UpdateRoutineHandler).Methods("PUT")
	r.HandleFunc("/api/v1/routines/{id}", c.DeleteRoutineHandler).Methods("DELETE")
	r.HandleFunc("/api/v1/routines/{id}/execute", c.ExecuteRoutineHandler).Methods("POST")
	return r
}

// --- tests ---

func TestCreateRoutineHandler_Success(t *testing.T) {
	repo := newMockRoutineRepo()
	router := makeRoutineRouter(repo, &mockRoutineExecutor{})

	body := map[string]interface{}{
		"name":    "Chegou em casa",
		"enabled": true,
		"trigger": map[string]interface{}{
			"type": "schedule",
			"cron": "0 22 * * *",
		},
		"actions": []map[string]interface{}{
			{"type": "device", "device_id": 1, "command": "on", "sort_order": 1},
		},
	}
	req := makeRoutineRequest("POST", "/api/v1/routines", body, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateRoutineHandler_MissingUserID(t *testing.T) {
	repo := newMockRoutineRepo()
	router := makeRoutineRouter(repo, &mockRoutineExecutor{})

	req := httptest.NewRequest("POST", "/api/v1/routines", bytes.NewBufferString(`{"name":"X"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestListRoutinesHandler(t *testing.T) {
	repo := newMockRoutineRepo()
	repo.CreateRoutine(&model.Routine{UserID: 1, Name: "A", Trigger: model.RoutineTrigger{Type: "schedule"}})
	repo.CreateRoutine(&model.Routine{UserID: 1, Name: "B", Trigger: model.RoutineTrigger{Type: "device"}})
	router := makeRoutineRouter(repo, &mockRoutineExecutor{})

	req := makeRoutineRequest("GET", "/api/v1/routines", nil, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestGetRoutineHandler_Found(t *testing.T) {
	repo := newMockRoutineRepo()
	r := &model.Routine{UserID: 1, Name: "Rotina Teste", Trigger: model.RoutineTrigger{Type: "schedule"}}
	id, _ := repo.CreateRoutine(r)
	router := makeRoutineRouter(repo, &mockRoutineExecutor{})

	req := makeRoutineRequest("GET", fmt.Sprintf("/api/v1/routines/%d", id), nil, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestGetRoutineHandler_NotFound(t *testing.T) {
	repo := newMockRoutineRepo()
	router := makeRoutineRouter(repo, &mockRoutineExecutor{})

	req := makeRoutineRequest("GET", "/api/v1/routines/999", nil, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetRoutineHandler_Forbidden(t *testing.T) {
	repo := newMockRoutineRepo()
	r := &model.Routine{UserID: 1, Name: "Rotina do user 1", Trigger: model.RoutineTrigger{Type: "schedule"}}
	id, _ := repo.CreateRoutine(r)
	router := makeRoutineRouter(repo, &mockRoutineExecutor{})

	req := makeRoutineRequest("GET", fmt.Sprintf("/api/v1/routines/%d", id), nil, 2)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestUpdateRoutineHandler_Success(t *testing.T) {
	repo := newMockRoutineRepo()
	r := &model.Routine{UserID: 1, Name: "Antiga", Trigger: model.RoutineTrigger{Type: "schedule"}}
	id, _ := repo.CreateRoutine(r)
	router := makeRoutineRouter(repo, &mockRoutineExecutor{})

	body := map[string]interface{}{
		"name":    "Nova",
		"enabled": true,
		"trigger": map[string]interface{}{"type": "schedule", "cron": "0 8 * * *"},
		"actions": []interface{}{},
	}
	req := makeRoutineRequest("PUT", fmt.Sprintf("/api/v1/routines/%d", id), body, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteRoutineHandler_Success(t *testing.T) {
	repo := newMockRoutineRepo()
	r := &model.Routine{UserID: 1, Name: "Para deletar", Trigger: model.RoutineTrigger{Type: "schedule"}}
	id, _ := repo.CreateRoutine(r)
	router := makeRoutineRouter(repo, &mockRoutineExecutor{})

	req := makeRoutineRequest("DELETE", fmt.Sprintf("/api/v1/routines/%d", id), nil, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestExecuteRoutineHandler_Success(t *testing.T) {
	repo := newMockRoutineRepo()
	r := &model.Routine{
		UserID:  1,
		Name:    "Rotina Manual",
		Enabled: true,
		Trigger: model.RoutineTrigger{Type: "schedule"},
		Actions: []model.RoutineAction{
			{Type: "device", DeviceID: uint64Ptr(10), Command: strPtr2("on"), SortOrder: 1},
		},
	}
	id, _ := repo.CreateRoutine(r)
	router := makeRoutineRouter(repo, &mockRoutineExecutor{})

	req := makeRoutineRequest("POST", fmt.Sprintf("/api/v1/routines/%d/execute", id), nil, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func uint64Ptr(v uint64) *uint64 { return &v }
func strPtr2(s string) *string   { return &s }

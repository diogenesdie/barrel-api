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

// --- mock scene repository ---

type mockSceneRepo struct {
	scenes map[uint64]*model.Scene
	nextID uint64
}

func newMockSceneRepo() *mockSceneRepo {
	return &mockSceneRepo{scenes: make(map[uint64]*model.Scene), nextID: 1}
}

func (m *mockSceneRepo) CreateScene(scene *model.Scene) (uint64, error) {
	id := m.nextID
	m.nextID++
	scene.ID = id
	scene.CreatedAt = time.Now()
	scene.UpdatedAt = time.Now()
	m.scenes[id] = scene
	return id, nil
}

func (m *mockSceneRepo) GetSceneByID(id uint64) (*model.Scene, error) {
	s, ok := m.scenes[id]
	if !ok {
		return nil, repository.ErrSceneNotFound
	}
	return s, nil
}

func (m *mockSceneRepo) GetScenesByUser(userID uint64) ([]model.Scene, error) {
	result := []model.Scene{}
	for _, s := range m.scenes {
		if s.UserID == userID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockSceneRepo) UpdateScene(scene *model.Scene) error {
	if _, ok := m.scenes[scene.ID]; !ok {
		return repository.ErrSceneNotFound
	}
	m.scenes[scene.ID] = scene
	return nil
}

func (m *mockSceneRepo) DeleteScene(id uint64, userID uint64) error {
	s, ok := m.scenes[id]
	if !ok {
		return repository.ErrSceneNotFound
	}
	if s.UserID != userID {
		return repository.ErrSceneNotFound
	}
	delete(m.scenes, id)
	return nil
}

// --- mock scene executor ---

type mockSceneExecutor struct{}

func (m *mockSceneExecutor) ExecuteScene(scene *model.Scene, userID uint64) (*model.SceneExecutionResult, error) {
	results := make([]model.SceneActionResult, len(scene.Actions))
	for i, a := range scene.Actions {
		results[i] = model.SceneActionResult{DeviceID: a.DeviceID, Command: a.Command, Success: true}
	}
	return &model.SceneExecutionResult{SceneID: scene.ID, Actions: results}, nil
}

// --- helpers ---

func makeSceneRequest(method, path string, body interface{}, userID uint64) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("user_id", fmt.Sprintf("%d", userID))
	return req
}

func makeSceneRouter(sceneRepo repository.SceneRepositoryInterface, executor controller.SceneExecutorInterface) *mux.Router {
	r := mux.NewRouter()
	c := controller.NewSceneController(sceneRepo, executor)
	r.HandleFunc("/api/v1/scenes", c.ListScenesHandler).Methods("GET")
	r.HandleFunc("/api/v1/scenes", c.CreateSceneHandler).Methods("POST")
	r.HandleFunc("/api/v1/scenes/{id}", c.GetSceneHandler).Methods("GET")
	r.HandleFunc("/api/v1/scenes/{id}", c.UpdateSceneHandler).Methods("PUT")
	r.HandleFunc("/api/v1/scenes/{id}", c.DeleteSceneHandler).Methods("DELETE")
	r.HandleFunc("/api/v1/scenes/{id}/execute", c.ExecuteSceneHandler).Methods("POST")
	return r
}

// --- tests ---

func TestCreateSceneHandler_Success(t *testing.T) {
	repo := newMockSceneRepo()
	router := makeSceneRouter(repo, &mockSceneExecutor{})

	body := map[string]interface{}{
		"name": "Modo Cinema",
		"actions": []map[string]interface{}{
			{"device_id": 1, "command": "off", "sort_order": 1},
		},
	}
	req := makeSceneRequest("POST", "/api/v1/scenes", body, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCreateSceneHandler_MissingUserID(t *testing.T) {
	repo := newMockSceneRepo()
	router := makeSceneRouter(repo, &mockSceneExecutor{})

	req := httptest.NewRequest("POST", "/api/v1/scenes", bytes.NewBufferString(`{"name":"X"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestListScenesHandler(t *testing.T) {
	repo := newMockSceneRepo()
	repo.CreateScene(&model.Scene{UserID: 1, Name: "Cena A"})
	repo.CreateScene(&model.Scene{UserID: 1, Name: "Cena B"})
	router := makeSceneRouter(repo, &mockSceneExecutor{})

	req := makeSceneRequest("GET", "/api/v1/scenes", nil, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestGetSceneHandler_Found(t *testing.T) {
	repo := newMockSceneRepo()
	scene := &model.Scene{UserID: 1, Name: "Cena Teste"}
	id, _ := repo.CreateScene(scene)
	router := makeSceneRouter(repo, &mockSceneExecutor{})

	req := makeSceneRequest("GET", fmt.Sprintf("/api/v1/scenes/%d", id), nil, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestGetSceneHandler_NotFound(t *testing.T) {
	repo := newMockSceneRepo()
	router := makeSceneRouter(repo, &mockSceneExecutor{})

	req := makeSceneRequest("GET", "/api/v1/scenes/999", nil, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetSceneHandler_Forbidden(t *testing.T) {
	repo := newMockSceneRepo()
	scene := &model.Scene{UserID: 1, Name: "Cena do user 1"}
	id, _ := repo.CreateScene(scene)
	router := makeSceneRouter(repo, &mockSceneExecutor{})

	// user 2 trying to access user 1's scene
	req := makeSceneRequest("GET", fmt.Sprintf("/api/v1/scenes/%d", id), nil, 2)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}

func TestUpdateSceneHandler_Success(t *testing.T) {
	repo := newMockSceneRepo()
	scene := &model.Scene{UserID: 1, Name: "Antiga"}
	id, _ := repo.CreateScene(scene)
	router := makeSceneRouter(repo, &mockSceneExecutor{})

	body := map[string]interface{}{"name": "Nova", "actions": []interface{}{}}
	req := makeSceneRequest("PUT", fmt.Sprintf("/api/v1/scenes/%d", id), body, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteSceneHandler_Success(t *testing.T) {
	repo := newMockSceneRepo()
	scene := &model.Scene{UserID: 1, Name: "Para deletar"}
	id, _ := repo.CreateScene(scene)
	router := makeSceneRouter(repo, &mockSceneExecutor{})

	req := makeSceneRequest("DELETE", fmt.Sprintf("/api/v1/scenes/%d", id), nil, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestExecuteSceneHandler_Success(t *testing.T) {
	repo := newMockSceneRepo()
	scene := &model.Scene{
		UserID: 1,
		Name:   "Modo Cinema",
		Actions: []model.SceneAction{
			{DeviceID: 10, Command: "off", SortOrder: 1},
		},
	}
	id, _ := repo.CreateScene(scene)
	router := makeSceneRouter(repo, &mockSceneExecutor{})

	req := makeSceneRequest("POST", fmt.Sprintf("/api/v1/scenes/%d/execute", id), nil, 1)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

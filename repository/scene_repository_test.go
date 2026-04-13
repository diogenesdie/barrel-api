package repository_test

import (
	"barrel-api/model"
	"barrel-api/repository"
	"errors"
	"testing"
	"time"
)

// mockSceneRepository implements SceneRepositoryInterface for unit testing.
type mockSceneRepository struct {
	scenes map[uint64]*model.Scene
	nextID uint64
}

func newMockSceneRepository() *mockSceneRepository {
	return &mockSceneRepository{
		scenes: make(map[uint64]*model.Scene),
		nextID: 1,
	}
}

func (m *mockSceneRepository) CreateScene(scene *model.Scene) (uint64, error) {
	id := m.nextID
	m.nextID++
	scene.ID = id
	scene.CreatedAt = time.Now()
	scene.UpdatedAt = time.Now()
	m.scenes[id] = scene
	return id, nil
}

func (m *mockSceneRepository) GetSceneByID(id uint64) (*model.Scene, error) {
	s, ok := m.scenes[id]
	if !ok {
		return nil, repository.ErrSceneNotFound
	}
	return s, nil
}

func (m *mockSceneRepository) GetScenesByUser(userID uint64) ([]model.Scene, error) {
	result := []model.Scene{}
	for _, s := range m.scenes {
		if s.UserID == userID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockSceneRepository) UpdateScene(scene *model.Scene) error {
	if _, ok := m.scenes[scene.ID]; !ok {
		return repository.ErrSceneNotFound
	}
	m.scenes[scene.ID] = scene
	return nil
}

func (m *mockSceneRepository) DeleteScene(id uint64, userID uint64) error {
	s, ok := m.scenes[id]
	if !ok {
		return repository.ErrSceneNotFound
	}
	if s.UserID != userID {
		return errors.New("forbidden")
	}
	delete(m.scenes, id)
	return nil
}

// Tests

func TestCreateScene(t *testing.T) {
	repo := newMockSceneRepository()
	scene := &model.Scene{
		UserID: 1,
		Name:   "Modo Cinema",
		Icon:   strPtr("movie"),
		Actions: []model.SceneAction{
			{DeviceID: 10, Command: "off", SortOrder: 1},
			{DeviceID: 20, Command: "on", SortOrder: 2},
		},
	}

	id, err := repo.CreateScene(scene)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestGetSceneByID_Found(t *testing.T) {
	repo := newMockSceneRepository()
	scene := &model.Scene{UserID: 1, Name: "Cena Teste"}
	id, _ := repo.CreateScene(scene)

	got, err := repo.GetSceneByID(id)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got.Name != "Cena Teste" {
		t.Errorf("expected name 'Cena Teste', got '%s'", got.Name)
	}
}

func TestGetSceneByID_NotFound(t *testing.T) {
	repo := newMockSceneRepository()

	_, err := repo.GetSceneByID(999)
	if err != repository.ErrSceneNotFound {
		t.Fatalf("expected ErrSceneNotFound, got: %v", err)
	}
}

func TestGetScenesByUser(t *testing.T) {
	repo := newMockSceneRepository()
	repo.CreateScene(&model.Scene{UserID: 1, Name: "Cena A"})
	repo.CreateScene(&model.Scene{UserID: 1, Name: "Cena B"})
	repo.CreateScene(&model.Scene{UserID: 2, Name: "Cena C"})

	scenes, err := repo.GetScenesByUser(1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(scenes) != 2 {
		t.Errorf("expected 2 scenes for user 1, got %d", len(scenes))
	}
}

func TestUpdateScene(t *testing.T) {
	repo := newMockSceneRepository()
	scene := &model.Scene{UserID: 1, Name: "Antiga"}
	id, _ := repo.CreateScene(scene)

	scene.ID = id
	scene.Name = "Nova"
	err := repo.UpdateScene(scene)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := repo.GetSceneByID(id)
	if got.Name != "Nova" {
		t.Errorf("expected name 'Nova', got '%s'", got.Name)
	}
}

func TestUpdateScene_NotFound(t *testing.T) {
	repo := newMockSceneRepository()
	err := repo.UpdateScene(&model.Scene{ID: 999, UserID: 1, Name: "X"})
	if err != repository.ErrSceneNotFound {
		t.Fatalf("expected ErrSceneNotFound, got: %v", err)
	}
}

func TestDeleteScene(t *testing.T) {
	repo := newMockSceneRepository()
	scene := &model.Scene{UserID: 1, Name: "Para deletar"}
	id, _ := repo.CreateScene(scene)

	err := repo.DeleteScene(id, 1)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	_, err = repo.GetSceneByID(id)
	if err != repository.ErrSceneNotFound {
		t.Fatal("expected scene to be deleted")
	}
}

func TestDeleteScene_NotFound(t *testing.T) {
	repo := newMockSceneRepository()
	err := repo.DeleteScene(999, 1)
	if err != repository.ErrSceneNotFound {
		t.Fatalf("expected ErrSceneNotFound, got: %v", err)
	}
}

func TestDeleteScene_Forbidden(t *testing.T) {
	repo := newMockSceneRepository()
	scene := &model.Scene{UserID: 1, Name: "Cena do usuário 1"}
	id, _ := repo.CreateScene(scene)

	err := repo.DeleteScene(id, 2)
	if err == nil {
		t.Fatal("expected error when deleting scene owned by another user")
	}
}

func strPtr(s string) *string { return &s }

package core

import (
	"barrel-api/model"
	"barrel-api/repository"
	"log"

	"github.com/robfig/cron/v3"
)

// RoutineRunnerIface is satisfied by RoutineExecutor and by test mocks.
type RoutineRunnerIface interface {
	ExecuteRoutine(routine *model.Routine) error
}

// RoutineScheduler manages cron jobs for time-triggered routines.
// It loads all enabled schedule-type routines at startup and keeps
// the cron registry in sync as routines are created, updated or deleted.
type RoutineScheduler struct {
	routineRepo repository.RoutineRepositoryInterface
	executor    RoutineRunnerIface
	cron        *cron.Cron
	jobIDs      map[uint64]cron.EntryID // routineID → cron entry ID
}

func NewRoutineScheduler(routineRepo repository.RoutineRepositoryInterface, executor RoutineRunnerIface) *RoutineScheduler {
	return &RoutineScheduler{
		routineRepo: routineRepo,
		executor:    executor,
		cron:        cron.New(),
		jobIDs:      make(map[uint64]cron.EntryID),
	}
}

// JobIDs returns a copy of the current cron entry map (routineID → entryID).
// Intended for testing and observability only.
func (s *RoutineScheduler) JobIDs() map[uint64]cron.EntryID {
	cp := make(map[uint64]cron.EntryID, len(s.jobIDs))
	for k, v := range s.jobIDs {
		cp[k] = v
	}
	return cp
}

// Start loads all enabled schedule routines from the database and begins the cron loop.
func (s *RoutineScheduler) Start() error {
	routines, err := s.routineRepo.GetEnabledRoutinesByTriggerType("schedule")
	if err != nil {
		return err
	}

	for _, r := range routines {
		if err := s.addJob(r.ID, r.UserID, *r.Trigger.Cron, r.Name); err != nil {
			log.Printf("[scheduler] failed to schedule routine %d (%s): %v", r.ID, r.Name, err)
		}
	}

	s.cron.Start()
	log.Printf("[scheduler] started with %d scheduled routine(s)", len(s.jobIDs))
	return nil
}

// Stop halts the cron scheduler gracefully.
func (s *RoutineScheduler) Stop() {
	s.cron.Stop()
}

// AddRoutine registers a new cron job for a schedule-type routine.
// Safe to call after Start().
func (s *RoutineScheduler) AddRoutine(routineID, userID uint64, cronExpr, name string) error {
	if _, exists := s.jobIDs[routineID]; exists {
		s.RemoveRoutine(routineID)
	}
	return s.addJob(routineID, userID, cronExpr, name)
}

// RemoveRoutine removes the cron job for the given routine ID.
func (s *RoutineScheduler) RemoveRoutine(routineID uint64) {
	if entryID, ok := s.jobIDs[routineID]; ok {
		s.cron.Remove(entryID)
		delete(s.jobIDs, routineID)
	}
}

func (s *RoutineScheduler) addJob(routineID, userID uint64, cronExpr, name string) error {
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		routine, err := s.routineRepo.GetRoutineByID(routineID)
		if err != nil {
			log.Printf("[scheduler] routine %d not found, skipping: %v", routineID, err)
			return
		}
		if !routine.Enabled {
			return
		}
		if err := s.executor.ExecuteRoutine(routine); err != nil {
			log.Printf("[scheduler] execution error (routine=%d %s): %v", routineID, name, err)
		}
	})
	if err != nil {
		return err
	}
	s.jobIDs[routineID] = entryID
	log.Printf("[scheduler] scheduled routine %d (%s) with cron '%s'", routineID, name, cronExpr)
	return nil
}

package service

import (
	"log/slog"
	"server-master/pkg/utils"

	"github.com/robfig/cron/v3"
)

// Task defines the interface for a background task.
type Task interface {
	Name() string
	Spec() string // Cron specification string
	Run()
}

// Initializer defines the optional interface for tasks that require setup before scheduling.
type Initializer interface {
	Init() error
}

// Cleaner defines the optional interface for tasks that require cleanup when the scheduler stops.
type Cleaner interface {
	Cleanup()
}

// CronService is a generic background task scheduler.
type CronService struct {
	cron    *cron.Cron
	taskIDs *utils.SafeMap[string, cron.EntryID]
	tasks   *utils.SafeMap[string, Task]
}

// NewCronService creates a new instance of CronService.
func NewCronService() *CronService {
	return &CronService{
		cron:    cron.New(),
		taskIDs: utils.NewSafeMap[string, cron.EntryID](),
		tasks:   utils.NewSafeMap[string, Task](),
	}
}

// AddTask registers a new task with the scheduler.
// If the task implements Initializer, its Init() method is called first.
func (s *CronService) AddTask(t Task) error {
	if i, ok := t.(Initializer); ok {
		if err := i.Init(); err != nil {
			return err
		}
	}

	id, err := s.cron.AddFunc(t.Spec(), t.Run)
	if err != nil {
		return err
	}

	s.taskIDs.Set(t.Name(), id)
	s.tasks.Set(t.Name(), t)
	slog.Info("Task scheduled", "name", t.Name(), "spec", t.Spec())
	return nil
}

// RemoveTask removes a task from the scheduler by name.
func (s *CronService) RemoveTask(name string) {
	if id, ok := s.taskIDs.Get(name); ok {
		s.cron.Remove(id)

		// Perform cleanup if implemented
		if t, ok := s.tasks.Get(name); ok {
			if c, ok := t.(Cleaner); ok {
				c.Cleanup()
			}
			s.tasks.Remove(name)
		}

		s.taskIDs.Remove(name)
		slog.Info("Task removed", "name", name)
	}
}

// Start begins the cron scheduler.
func (s *CronService) Start() {
	s.cron.Start()
	slog.Info("Cron scheduler started")
}

// Stop halts the cron scheduler and performs cleanup for all tasks.
func (s *CronService) Stop() {
	s.cron.Stop()
	slog.Info("Cron scheduler stopped")

	s.tasks.Range(func(name string, t Task) bool {
		if c, ok := t.(Cleaner); ok {
			slog.Debug("Cleaning up task", "name", name)
			c.Cleanup()
		}
		return true
	})
}

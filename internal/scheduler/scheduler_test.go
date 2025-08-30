package scheduler

import (
	"testing"
	"time"

	"github.com/korjavin/tw2dynalist/internal/logger"
)

func TestSimpleScheduler(t *testing.T) {
	log := logger.New("DEBUG")
	taskExecuted := make(chan bool, 1)

	task := func() {
		taskExecuted <- true
	}

	scheduler := NewSimpleScheduler(100*time.Millisecond, task, log)

	// Test that the task is executed immediately
	go scheduler.Start()
	select {
	case <-taskExecuted:
		// success
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Task was not executed immediately")
	}

	// Test that the task is executed again after the interval
	select {
	case <-taskExecuted:
		// success
	case <-time.After(150 * time.Millisecond):
		t.Fatal("Task was not executed after the interval")
	}

	// Test that the scheduler can be stopped
	scheduler.Stop()
	// a bit of time for the scheduler to stop
	time.Sleep(50 * time.Millisecond)

	// Make sure the task is not executed again
	select {
	case <-taskExecuted:
		t.Fatal("Task was executed after the scheduler was stopped")
	case <-time.After(150 * time.Millisecond):
		// success
	}
}

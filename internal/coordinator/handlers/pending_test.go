package handlers

import (
	"context"
	"testing"

	"github.com/runabol/tork"
	"github.com/runabol/tork/broker"
	"github.com/runabol/tork/datastore/inmemory"
	"github.com/runabol/tork/internal/uuid"
	"github.com/runabol/tork/middleware/task"
	"github.com/stretchr/testify/assert"
)

func Test_handlePendingTask(t *testing.T) {
	ctx := context.Background()
	b := broker.NewInMemoryBroker()

	processed := make(chan any)
	err := b.SubscribeForTasks("test-queue", func(t *tork.Task) error {
		close(processed)
		return nil
	})
	assert.NoError(t, err)

	ds := inmemory.NewInMemoryDatastore()
	handler := NewPendingHandler(ds, b)
	assert.NotNil(t, handler)

	j1 := &tork.Job{
		ID:   uuid.NewUUID(),
		Name: "test job",
	}
	err = ds.CreateJob(ctx, j1)
	assert.NoError(t, err)

	tk := &tork.Task{
		ID:    uuid.NewUUID(),
		Queue: "test-queue",
		JobID: j1.ID,
	}

	err = ds.CreateTask(ctx, tk)
	assert.NoError(t, err)

	err = handler(ctx, task.StateChange, tk)
	assert.NoError(t, err)

	// wait for the task to get processed
	<-processed

	tk, err = ds.GetTaskByID(ctx, tk.ID)
	assert.NoError(t, err)
	assert.Equal(t, tork.TaskStateScheduled, tk.State)
}

func Test_handleConditionalTask(t *testing.T) {
	ctx := context.Background()
	b := broker.NewInMemoryBroker()

	completed := make(chan any)
	err := b.SubscribeForTasks(broker.QUEUE_COMPLETED, func(t *tork.Task) error {
		close(completed)
		return nil
	})
	assert.NoError(t, err)

	ds := inmemory.NewInMemoryDatastore()
	handler := NewPendingHandler(ds, b)
	assert.NotNil(t, handler)

	tk := &tork.Task{
		ID:    uuid.NewUUID(),
		Queue: "test-queue",
		If:    "false",
	}

	err = ds.CreateTask(ctx, tk)
	assert.NoError(t, err)

	err = handler(ctx, task.StateChange, tk)
	assert.NoError(t, err)

	// wait for the task to get processed
	<-completed

	tk, err = ds.GetTaskByID(ctx, tk.ID)
	assert.NoError(t, err)
	assert.Equal(t, tork.TaskStateSkipped, tk.State)
}

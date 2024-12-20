package inmemory_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/runabol/tork"
	"github.com/runabol/tork/datastore"
	"github.com/runabol/tork/datastore/inmemory"

	"github.com/runabol/tork/internal/uuid"
	"github.com/stretchr/testify/assert"
)

func TestInMemoryCreateAndGetTask(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	t1 := tork.Task{
		ID: uuid.NewUUID(),
	}
	err := ds.CreateTask(ctx, &t1)
	assert.NoError(t, err)
	t2, err := ds.GetTaskByID(ctx, t1.ID)
	assert.NoError(t, err)
	assert.Equal(t, t1.ID, t2.ID)
}

func TestInMemoryCreateJob(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	now := time.Now().UTC()
	u := &tork.User{
		ID:        uuid.NewUUID(),
		Username:  uuid.NewShortUUID(),
		Name:      "Tester",
		CreatedAt: &now,
	}
	err := ds.CreateUser(ctx, u)
	assert.NoError(t, err)
	j1 := tork.Job{
		ID:        uuid.NewUUID(),
		CreatedBy: u,
	}
	err = ds.CreateJob(ctx, &j1)
	assert.NoError(t, err)
	assert.Equal(t, u.Username, j1.CreatedBy.Username)

	j2, err := ds.GetJobByID(ctx, j1.ID)
	assert.NoError(t, err)
	assert.Equal(t, u.Username, j2.CreatedBy.Username)
}

func TestInMemoryGetActiveTasks(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	jid := uuid.NewUUID()

	tasks := []tork.Task{{
		ID:    uuid.NewUUID(),
		State: tork.TaskStatePending,
		JobID: jid,
	}, {
		ID:    uuid.NewUUID(),
		State: tork.TaskStateScheduled,
		JobID: jid,
	}, {
		ID:    uuid.NewUUID(),
		State: tork.TaskStateRunning,
		JobID: jid,
	}, {
		ID:    uuid.NewUUID(),
		State: tork.TaskStateCancelled,
		JobID: jid,
	}, {
		ID:    uuid.NewUUID(),
		State: tork.TaskStateCompleted,
		JobID: jid,
	}, {
		ID:    uuid.NewUUID(),
		State: tork.TaskStateFailed,
		JobID: jid,
	}}

	for _, ta := range tasks {
		err := ds.CreateTask(ctx, &ta)
		assert.NoError(t, err)
	}
	at, err := ds.GetActiveTasks(ctx, jid)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(at))
}

func TestInMemoryUpdateTask(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	t1 := tork.Task{
		ID:    uuid.NewUUID(),
		State: tork.TaskStatePending,
	}
	err := ds.CreateTask(ctx, &t1)
	assert.NoError(t, err)

	err = ds.UpdateTask(ctx, t1.ID, func(u *tork.Task) error {
		u.State = tork.TaskStateScheduled
		return nil
	})
	assert.NoError(t, err)

	t2, err := ds.GetTaskByID(ctx, t1.ID)
	assert.NoError(t, err)
	assert.Equal(t, tork.TaskStateScheduled, t2.State)
}

func TestInMemoryUpdateTaskConcurrently(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()

	now := time.Now().UTC()
	j1 := tork.Job{
		ID: uuid.NewUUID(),
	}
	err := ds.CreateJob(ctx, &j1)
	assert.NoError(t, err)
	t1 := &tork.Task{
		ID:        uuid.NewUUID(),
		CreatedAt: &now,
		JobID:     j1.ID,
		Parallel:  &tork.ParallelTask{},
		Env:       make(map[string]string),
	}
	err = ds.CreateTask(ctx, t1)
	assert.NoError(t, err)

	w := sync.WaitGroup{}
	w.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer w.Done()
			err := ds.UpdateTask(ctx, t1.ID, func(u *tork.Task) error {
				time.Sleep(time.Duration(rand.Intn(1000)) * time.Microsecond)
				u.State = tork.TaskStateScheduled
				u.Result = "my result"
				u.Parallel.Completions = u.Parallel.Completions + 1
				u.Env[fmt.Sprintf("SOME_VAR_%d", rand.Intn(100000))] = "some value"
				return nil
			})
			assert.NoError(t, err)
		}()
	}

	r := sync.WaitGroup{}
	r.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer r.Done()
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Microsecond)
			t2, err := ds.GetTaskByID(ctx, t1.ID)
			assert.NoError(t, err)
			_ = t2.Clone()
		}()
	}

	r.Wait()
	w.Wait()

	t2, err := ds.GetTaskByID(ctx, t1.ID)
	assert.NoError(t, err)
	assert.Equal(t, tork.TaskStateScheduled, t2.State)
	assert.Equal(t, "my result", t2.Result)
	assert.Equal(t, 1000, t2.Parallel.Completions)
}

func TestInMemoryUpdateJobConcurrently(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()

	j1 := tork.Job{
		ID:        uuid.NewUUID(),
		TaskCount: 0,
	}
	err := ds.CreateJob(ctx, &j1)
	assert.NoError(t, err)

	w := sync.WaitGroup{}
	w.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer w.Done()
			err := ds.UpdateJob(ctx, j1.ID, func(u *tork.Job) error {
				time.Sleep(time.Duration(rand.Intn(1000)) * time.Microsecond)
				u.TaskCount = u.TaskCount + 1
				if u.Context.Tasks == nil {
					u.Context.Tasks = make(map[string]string)
				}
				u.Context.Tasks[fmt.Sprintf("someVar-%d", rand.Intn(100000))] = "some value"
				return nil
			})
			assert.NoError(t, err)
		}()
	}

	r := sync.WaitGroup{}
	r.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer r.Done()
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Microsecond)
			j2, err := ds.GetJobByID(ctx, j1.ID)
			assert.NoError(t, err)
			_ = j2.Clone()
		}()
	}
	r.Wait()

	w.Wait()

	j2, err := ds.GetJobByID(ctx, j1.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1000, j2.TaskCount)
}

func TestInMemoryCreateAndGetNode(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	n1 := &tork.Node{
		ID: uuid.NewUUID(),
	}
	err := ds.CreateNode(ctx, n1)
	assert.NoError(t, err)
	n2, err := ds.GetNodeByID(ctx, n1.ID)
	assert.NoError(t, err)
	assert.Equal(t, n1.ID, n2.ID)
}

func TestInMemoryUpdateNode(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	n1 := &tork.Node{
		ID:              uuid.NewUUID(),
		LastHeartbeatAt: time.Now().UTC().Add(-time.Minute),
	}
	err := ds.CreateNode(ctx, n1)
	assert.NoError(t, err)

	now := time.Now().UTC()

	err = ds.UpdateNode(ctx, n1.ID, func(u *tork.Node) error {
		u.LastHeartbeatAt = now
		return nil
	})
	assert.NoError(t, err)

	n2, err := ds.GetNodeByID(ctx, n1.ID)
	assert.NoError(t, err)
	assert.Equal(t, now, n2.LastHeartbeatAt)
}

func TestInMemoryUpdateNodeConcurrently(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	n1 := &tork.Node{
		ID:              uuid.NewUUID(),
		LastHeartbeatAt: time.Now().UTC().Add(-time.Minute),
	}
	err := ds.CreateNode(ctx, n1)
	assert.NoError(t, err)

	w := sync.WaitGroup{}
	w.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer w.Done()
			err := ds.UpdateNode(ctx, n1.ID, func(u *tork.Node) error {
				time.Sleep(time.Duration(rand.Intn(1000)) * time.Microsecond)
				u.TaskCount = u.TaskCount + 1
				return nil
			})
			assert.NoError(t, err)
		}()
	}

	r := sync.WaitGroup{}
	r.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer r.Done()
			time.Sleep(time.Duration(rand.Intn(1000)) * time.Microsecond)
			n2, err := ds.GetNodeByID(ctx, n1.ID)
			assert.NoError(t, err)
			_ = n2.Clone()
		}()
	}

	r.Wait()
	w.Wait()

	n2, err := ds.GetNodeByID(ctx, n1.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1000, n2.TaskCount)
}

func TestInMemoryExpiredNodes(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore(
		inmemory.WithCleanupInterval(time.Millisecond*20),
		inmemory.WithNodeExpiration(time.Millisecond*10),
	)
	n := &tork.Node{
		ID: uuid.NewUUID(),
	}
	err := ds.CreateNode(ctx, n)
	assert.NoError(t, err)
	n1, err := ds.GetNodeByID(ctx, n.ID)
	assert.NoError(t, err)
	assert.Equal(t, n.ID, n1.ID)
	time.Sleep(time.Millisecond * 100)
	_, err = ds.GetNodeByID(ctx, n.ID)
	assert.ErrorIs(t, err, datastore.ErrNodeNotFound)
}

func TestInMemoryExpiredJob(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore(
		inmemory.WithCleanupInterval(time.Millisecond*20),
		inmemory.WithJobExpiration(time.Millisecond*10),
	)
	j := &tork.Job{
		ID:    uuid.NewUUID(),
		Name:  "test job",
		State: tork.JobStateRunning,
	}
	err := ds.CreateJob(ctx, j)
	assert.NoError(t, err)

	ta := &tork.Task{
		ID:    uuid.NewUUID(),
		Name:  "test task",
		JobID: j.ID,
	}
	err = ds.CreateTask(ctx, ta)
	assert.NoError(t, err)

	j1, err := ds.GetJobByID(ctx, j.ID)
	assert.NoError(t, err)
	assert.Equal(t, j.ID, j1.ID)

	t1, err := ds.GetTaskByID(ctx, ta.ID)
	assert.NoError(t, err)
	assert.Equal(t, ta.ID, t1.ID)

	time.Sleep(time.Millisecond * 100)

	// should not be evicted yet --
	// as the job is still running
	j1, err = ds.GetJobByID(ctx, j.ID)
	assert.NoError(t, err)
	assert.Equal(t, j.ID, j1.ID)

	t1, err = ds.GetTaskByID(ctx, ta.ID)
	assert.NoError(t, err)
	assert.Equal(t, ta.ID, t1.ID)

	// completing the job
	err = ds.UpdateJob(ctx, j.ID, func(u *tork.Job) error {
		u.State = tork.JobStateCompleted
		return nil
	})
	assert.NoError(t, err)

	time.Sleep(time.Second * 1)

	// should be evicted now
	_, err = ds.GetJobByID(ctx, j.ID)
	assert.ErrorIs(t, err, datastore.ErrJobNotFound)

	_, err = ds.GetTaskByID(ctx, ta.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, datastore.ErrTaskNotFound)
}

func TestInMemoryCreateAndGetTaskLogs(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	t1 := tork.Task{
		ID: uuid.NewUUID(),
	}
	err := ds.CreateTask(ctx, &t1)
	assert.NoError(t, err)

	err = ds.CreateTaskLogPart(ctx, &tork.TaskLogPart{
		Number:   1,
		TaskID:   t1.ID,
		Contents: "line 1",
	})
	assert.NoError(t, err)

	logs, err := ds.GetTaskLogParts(ctx, t1.ID, "", 1, 10)
	assert.NoError(t, err)
	assert.Len(t, logs.Items, 1)
	assert.Equal(t, "line 1", logs.Items[0].Contents)
	assert.Equal(t, 1, logs.TotalPages)
}

func TestInMemoryCreateAndGetTaskLogsMultiParts(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	t1 := tork.Task{
		ID: uuid.NewUUID(),
	}
	err := ds.CreateTask(ctx, &t1)
	assert.NoError(t, err)

	parts := 10

	wg := sync.WaitGroup{}
	wg.Add(parts)

	for i := 1; i <= parts; i++ {
		go func(n int) {
			defer wg.Done()
			err := ds.CreateTaskLogPart(ctx, &tork.TaskLogPart{
				Number:   n,
				TaskID:   t1.ID,
				Contents: fmt.Sprintf("line %d", n),
			})
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	logs, err := ds.GetTaskLogParts(ctx, t1.ID, "", 1, 10)
	assert.NoError(t, err)
	assert.Len(t, logs.Items, 10)
	assert.Equal(t, "line 10", logs.Items[0].Contents)
	assert.Equal(t, "line 1", logs.Items[9].Contents)
}

func TestInMemoryCreateAndGetTaskLogsLarge(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	t1 := tork.Task{
		ID: uuid.NewUUID(),
	}
	err := ds.CreateTask(ctx, &t1)
	assert.NoError(t, err)

	for i := 1; i <= 100; i++ {
		err = ds.CreateTaskLogPart(ctx, &tork.TaskLogPart{
			Number:   i,
			TaskID:   t1.ID,
			Contents: fmt.Sprintf("line %d", i),
		})
		assert.NoError(t, err)
	}

	logs, err := ds.GetTaskLogParts(ctx, t1.ID, "", 1, 10)
	assert.NoError(t, err)
	assert.Len(t, logs.Items, 10)
	assert.Equal(t, "line 100", logs.Items[0].Contents)
	assert.Equal(t, "line 91", logs.Items[9].Contents)
	assert.Equal(t, 10, logs.Size)
	assert.Equal(t, 10, logs.TotalPages)
}

func TestInMemoryGetJobLogParts(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	jid := uuid.NewUUID()
	t1 := tork.Task{
		ID:    uuid.NewUUID(),
		JobID: jid,
	}
	err := ds.CreateTask(ctx, &t1)
	assert.NoError(t, err)

	logs, err := ds.GetJobLogParts(ctx, jid, "", 1, 10)
	assert.NoError(t, err)
	assert.Len(t, logs.Items, 0)

	err = ds.CreateTaskLogPart(ctx, &tork.TaskLogPart{
		Number:   1,
		TaskID:   t1.ID,
		Contents: "line 1",
	})
	assert.NoError(t, err)

	logs, err = ds.GetJobLogParts(ctx, jid, "", 1, 10)
	assert.NoError(t, err)
	assert.Len(t, logs.Items, 1)
	assert.Equal(t, "line 1", logs.Items[0].Contents)
	assert.Equal(t, 1, logs.TotalPages)

	err = ds.CreateTaskLogPart(ctx, &tork.TaskLogPart{
		Number:   2,
		TaskID:   t1.ID,
		Contents: "line 2",
	})
	assert.NoError(t, err)

	logs, err = ds.GetJobLogParts(ctx, jid, "", 1, 10)
	assert.NoError(t, err)
	assert.Len(t, logs.Items, 2)
	assert.Equal(t, "line 2", logs.Items[0].Contents)
	assert.Equal(t, 1, logs.TotalPages)
}

func TestInMemoryCreateAndQueryTaskLogPartsLarge(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	t1 := tork.Task{
		ID: uuid.NewUUID(),
	}
	err := ds.CreateTask(ctx, &t1)
	assert.NoError(t, err)

	for i := 1; i <= 100; i++ {
		err = ds.CreateTaskLogPart(ctx, &tork.TaskLogPart{
			Number:   i,
			TaskID:   t1.ID,
			Contents: fmt.Sprintf("line %d", i),
		})
		assert.NoError(t, err)
	}

	logs, err := ds.GetTaskLogParts(ctx, t1.ID, "100", 1, 10)
	assert.NoError(t, err)
	assert.Len(t, logs.Items, 1)
	assert.Equal(t, "line 100", logs.Items[0].Contents)
	assert.Equal(t, 1, logs.Size)
	assert.Equal(t, 1, logs.TotalPages)
}

func TestInMemoryCreateAndQueryJobLogPartsLarge(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	jid := uuid.NewUUID()
	t1 := tork.Task{
		ID:    uuid.NewUUID(),
		JobID: jid,
	}
	err := ds.CreateTask(ctx, &t1)
	assert.NoError(t, err)

	logs, err := ds.GetJobLogParts(ctx, jid, "", 1, 10)
	assert.NoError(t, err)
	assert.Len(t, logs.Items, 0)

	for i := 1; i <= 100; i++ {
		err = ds.CreateTaskLogPart(ctx, &tork.TaskLogPart{
			Number:   2,
			TaskID:   t1.ID,
			Contents: fmt.Sprintf("line %d", i),
		})
		assert.NoError(t, err)
	}

	logs, err = ds.GetJobLogParts(ctx, jid, "line 99", 1, 10)
	assert.NoError(t, err)
	assert.Len(t, logs.Items, 1)
	assert.Equal(t, "line 99", logs.Items[0].Contents)
	assert.Equal(t, 1, logs.TotalPages)
}

func TestInMemorySearchJobs(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()

	u1 := &tork.User{
		ID:       uuid.NewUUID(),
		Username: uuid.NewShortUUID(),
		Name:     "Tester",
	}
	err := ds.CreateUser(ctx, u1)
	assert.NoError(t, err)

	u2 := &tork.User{
		ID:       uuid.NewUUID(),
		Username: uuid.NewShortUUID(),
		Name:     "Tester",
	}
	err = ds.CreateUser(ctx, u2)
	assert.NoError(t, err)

	r := &tork.Role{
		Slug: "test-role",
		Name: "Test Role",
	}
	err = ds.CreateRole(ctx, r)
	assert.NoError(t, err)

	err = ds.AssignRole(ctx, u2.ID, r.ID)
	assert.NoError(t, err)

	u3 := &tork.User{
		ID:       uuid.NewUUID(),
		Username: uuid.NewShortUUID(),
		Name:     "Tester",
	}
	err = ds.CreateUser(ctx, u3)
	assert.NoError(t, err)

	for i := 0; i < 100; i++ {
		j1 := tork.Job{
			ID:    uuid.NewUUID(),
			Name:  fmt.Sprintf("Job %d", (i + 1)),
			State: tork.JobStateRunning,
			Tasks: []*tork.Task{{
				Name: "some task",
			}},
			Tags: []string{fmt.Sprintf("tag-%d", i)},
			Permissions: []*tork.Permission{{
				User: u1,
			}, {
				Role: r,
			}},
		}
		err := ds.CreateJob(ctx, &j1)
		assert.NoError(t, err)

		now := time.Now().UTC()
		err = ds.CreateTask(ctx, &tork.Task{
			ID:        uuid.NewUUID(),
			JobID:     j1.ID,
			State:     tork.TaskStateRunning,
			CreatedAt: &now,
		})
		assert.NoError(t, err)
	}

	for i := 100; i < 101; i++ {
		j1 := tork.Job{
			ID:    uuid.NewUUID(),
			Name:  fmt.Sprintf("Job %d", (i + 1)),
			State: tork.JobStateRunning,
			Tasks: []*tork.Task{{
				Name: "some task",
			}},
			Tags: []string{fmt.Sprintf("tag-%d", i)},
		}
		err := ds.CreateJob(ctx, &j1)
		assert.NoError(t, err)

		now := time.Now().UTC()
		err = ds.CreateTask(ctx, &tork.Task{
			ID:        uuid.NewUUID(),
			JobID:     j1.ID,
			State:     tork.TaskStateRunning,
			CreatedAt: &now,
		})
		assert.NoError(t, err)
	}

	p1, err := ds.GetJobs(ctx, "", "", 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, 10, p1.Size)
	assert.Equal(t, 101, p1.TotalItems)

	p1, err = ds.GetJobs(ctx, "", "101", 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, 1, p1.Size)
	assert.Equal(t, 1, p1.TotalItems)

	p1, err = ds.GetJobs(ctx, "", "tag:tag-1", 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, 1, p1.Size)
	assert.Equal(t, 1, p1.TotalItems)

	p1, err = ds.GetJobs(ctx, "", "tag:not-a-tag", 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, 0, p1.Size)
	assert.Equal(t, 0, p1.TotalItems)

	p1, err = ds.GetJobs(ctx, "", "tags:not-a-tag,tag-1", 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, 1, p1.Size)
	assert.Equal(t, 1, p1.TotalItems)

	p1, err = ds.GetJobs(ctx, "", "Job", 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, 10, p1.Size)
	assert.Equal(t, 101, p1.TotalItems)

	p1, err = ds.GetJobs(ctx, "", "running", 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, 10, p1.Size)
	assert.Equal(t, 101, p1.TotalItems)

	p1, err = ds.GetJobs(ctx, u1.Username, "running", 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, 10, p1.Size)
	assert.Equal(t, 101, p1.TotalItems)

	p1, err = ds.GetJobs(ctx, u2.Username, "running", 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, 10, p1.Size)
	assert.Equal(t, 101, p1.TotalItems)

	p1, err = ds.GetJobs(ctx, u3.Username, "running", 1, 10)
	assert.NoError(t, err)
	assert.Equal(t, 1, p1.Size)
	assert.Equal(t, 1, p1.TotalItems)
}

func TestInMemoryCreateRole(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	now := time.Now().UTC()
	r := &tork.Role{
		ID:        uuid.NewUUID(),
		Slug:      "test-role",
		Name:      "Test Role",
		CreatedAt: &now,
	}
	err := ds.CreateRole(ctx, r)
	assert.NoError(t, err)

	role, err := ds.GetRole(ctx, r.Slug)
	assert.NoError(t, err)
	assert.Equal(t, r.Slug, role.Slug)

	roles, err := ds.GetRoles(ctx)
	assert.NoError(t, err)
	assert.Len(t, roles, 1)
	assert.Equal(t, "Test Role", roles[0].Name)

	u := &tork.User{
		ID:        uuid.NewUUID(),
		Username:  uuid.NewShortUUID(),
		Name:      "Tester",
		CreatedAt: &now,
	}
	err = ds.CreateUser(ctx, u)
	assert.NoError(t, err)

	err = ds.AssignRole(ctx, u.ID, r.ID)
	assert.NoError(t, err)

	uroles, err := ds.GetUserRoles(ctx, u.ID)
	assert.NoError(t, err)
	assert.Len(t, uroles, 1)
	assert.Equal(t, r.ID, uroles[0].ID)

	err = ds.UnassignRole(ctx, u.ID, r.ID)
	assert.NoError(t, err)

	uroles, err = ds.GetUserRoles(ctx, u.ID)
	assert.NoError(t, err)
	assert.Len(t, uroles, 0)
}

func TestInMemoryGetNextTask(t *testing.T) {
	ctx := context.Background()
	ds := inmemory.NewInMemoryDatastore()
	jid := uuid.NewUUID()

	tasks := []tork.Task{{
		ID:    "parent-id",
		State: tork.TaskStatePending,
		JobID: jid,
	}, {
		ID:       "child-id",
		ParentID: "parent-id",
		State:    tork.TaskStateCreated,
		JobID:    jid,
	}}

	for _, ta := range tasks {
		err := ds.CreateTask(ctx, &ta)
		assert.NoError(t, err)
	}
	nt, err := ds.GetNextTask(ctx, "parent-id")
	assert.NoError(t, err)
	assert.Equal(t, "child-id", nt.ID)

	_, err = ds.GetNextTask(ctx, "no-such-id")
	assert.Error(t, err)
}

package runtime

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/tork/task"
)

type DockerRuntime struct {
	client *client.Client
	tasks  map[string]string
	images map[string]bool
	mu     sync.RWMutex
}

func NewDockerRuntime() (*DockerRuntime, error) {
	dc, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &DockerRuntime{
		client: dc,
		tasks:  make(map[string]string),
		images: make(map[string]bool),
		mu:     sync.RWMutex{},
	}, nil
}

type filteredReader struct {
	reader io.Reader
}

func (r filteredReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if err != nil {
		return n, err
	}
	j := 0
	for i := 0; i < n; i++ {
		if p[i] != 0 && // null
			p[i] != 1 && // start of heading
			p[i] != 11 { // tab
			p[j] = p[i]
			j++
		}
	}
	if j == 0 {
		return 0, io.EOF
	}
	return j, nil
}

func (d *DockerRuntime) imagePull(ctx context.Context, t task.Task) error {
	d.mu.RLock()
	_, ok := d.images[t.Image]
	d.mu.RUnlock()
	if ok {
		return nil
	}
	// let's check if we have the image
	// locally already
	images, err := d.client.ImageList(
		ctx,
		types.ImageListOptions{All: true},
	)
	if err != nil {
		return err
	}
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == t.Image {
				d.mu.Lock()
				d.images[tag] = true
				d.mu.Unlock()
				return nil
			}
		}
	}
	// this is intended. we don't want to pull
	// more than one image at a time to prevent
	// from saturating the nw inteface and to play
	// nice with the docker registry.
	d.mu.Lock()
	defer d.mu.Unlock()
	reader, err := d.client.ImagePull(
		ctx, t.Image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}
	return nil
}

func (d *DockerRuntime) Run(ctx context.Context, t task.Task) (string, error) {
	if err := d.imagePull(ctx, t); err != nil {
		return "", errors.Wrapf(err, "error pulling image")
	}

	env := []string{}
	for name, value := range t.Env {
		env = append(env, fmt.Sprintf("%s=%s", name, value))
	}

	var mounts []mount.Mount

	for _, v := range t.Volumes {
		vol := strings.Split(v, ":")
		if len(vol) != 2 {
			return "", errors.Errorf("invalid volume name: %s", v)
		}
		mount := mount.Mount{
			Type:   mount.TypeBind,
			Source: vol[0],
			Target: vol[1],
		}
		mounts = append(mounts, mount)
	}

	cpus, err := parseCPUs(t.Limits.CPUs)
	if err != nil {
		return "", errors.Wrapf(err, "invalid CPUs value")
	}

	mem, err := parseMemory(t.Limits.Memory)
	if err != nil {
		return "", errors.Wrapf(err, "invalid memory value")
	}

	hc := container.HostConfig{
		PublishAllPorts: true,
		Mounts:          mounts,
		Resources: container.Resources{
			NanoCPUs: cpus,
			Memory:   mem,
		},
	}
	cmd := t.CMD
	if len(cmd) == 0 {
		cmd = []string{"/tork/run"}
	}
	entrypoint := t.Entrypoint
	if len(entrypoint) == 0 && t.Run != "" {
		entrypoint = []string{"sh", "-c"}
	}
	cc := container.Config{
		Image:      t.Image,
		Env:        env,
		Cmd:        cmd,
		Entrypoint: entrypoint,
	}

	resp, err := d.client.ContainerCreate(
		ctx, &cc, &hc, nil, nil, "")
	if err != nil {
		log.Error().Msgf(
			"Error creating container using image %s: %v\n",
			t.Image, err,
		)
		return "", err
	}

	d.mu.Lock()
	d.tasks[t.ID] = resp.ID
	d.mu.Unlock()

	err = d.client.ContainerStart(
		ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "error starting container %s: %v\n", resp.ID, err)
	}

	// remove the container
	defer func() {
		if err := d.Stop(ctx, t); err != nil {
			log.Error().
				Err(err).
				Str("container-id", resp.ID).
				Msg("error removing container upon completion")
		}
	}()

	out, err := d.client.ContainerLogs(
		ctx,
		resp.ID,
		types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		},
	)
	if err != nil {
		return "", errors.Wrapf(err, "error getting logs for container %s: %v\n", resp.ID, err)
	}
	stdout := filteredReader{reader: out}
	defer func() {
		if err := out.Close(); err != nil {
			log.Error().Err(err).Msgf("error closing stdout on container %s", resp.ID)
		}
	}()
	// limit the amount of data read from stdout to prevent memory exhaustion
	lr := &io.LimitedReader{R: stdout, N: 1024}
	bufout := new(strings.Builder)
	tee := io.TeeReader(lr, bufout)
	_, err = io.Copy(os.Stdout, tee)
	if err != nil {
		return "", errors.Wrapf(err, "error reading the std out")
	}
	statusCh, errCh := d.client.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", err
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return "", errors.Errorf("exit code %d: %s", status.StatusCode, bufout.String())
		}
		log.Debug().
			Int64("status-code", status.StatusCode).
			Str("task-id", t.ID).
			Msg("task completed")
	}

	return bufout.String(), nil
}

func (d *DockerRuntime) Stop(ctx context.Context, t task.Task) error {
	d.mu.RLock()
	containerID, ok := d.tasks[t.ID]
	d.mu.RUnlock()
	if !ok {
		return nil
	}
	d.mu.Lock()
	delete(d.tasks, t.ID)
	d.mu.Unlock()
	log.Printf("Attempting to stop and remove container %v", containerID)
	return d.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		RemoveVolumes: false,
		RemoveLinks:   false,
		Force:         true,
	})
}

// take from https://github.com/docker/cli/blob/9bd5ec504afd13e82d5e50b60715e7190c1b2aa0/opts/opts.go#L393-L403
func parseCPUs(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	cpu, ok := new(big.Rat).SetString(value)
	if !ok {
		return 0, fmt.Errorf("failed to parse %v as a rational number", value)
	}
	nano := cpu.Mul(cpu, big.NewRat(1e9, 1))
	if !nano.IsInt() {
		return 0, fmt.Errorf("value is too precise")
	}
	return nano.Num().Int64(), nil
}

func parseMemory(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	return units.RAMInBytes(value)
}

package input

import (
	"github.com/runabol/tork"
	"golang.org/x/exp/maps"
)

type Task struct {
	Name        string            `json:"name,omitempty" yaml:"name,omitempty" validate:"required"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	CMD         []string          `json:"cmd,omitempty" yaml:"cmd,omitempty"`
	Entrypoint  []string          `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	Run         string            `json:"run,omitempty" yaml:"run,omitempty"`
	Image       string            `json:"image,omitempty" yaml:"image,omitempty"`
	Registry    *Registry         `json:"registry,omitempty" yaml:"registry,omitempty"`
	Env         map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Files       map[string]string `json:"files,omitempty" yaml:"files,omitempty"`
	Queue       string            `json:"queue,omitempty" yaml:"queue,omitempty" validate:"queue"`
	Pre         []AuxTask         `json:"pre,omitempty" yaml:"pre,omitempty" validate:"dive"`
	Post        []AuxTask         `json:"post,omitempty" yaml:"post,omitempty" validate:"dive"`
	Sidecars    []SidecarTask     `json:"sidecars,omitempty" yaml:"sidecars,omitempty" validate:"dive"`
	Mounts      []Mount           `json:"mounts,omitempty" yaml:"mounts,omitempty" validate:"dive"`
	Networks    []string          `json:"networks,omitempty" yaml:"networks,omitempty"`
	Retry       *Retry            `json:"retry,omitempty" yaml:"retry,omitempty"`
	Limits      *Limits           `json:"limits,omitempty" yaml:"limits,omitempty"`
	Timeout     string            `json:"timeout,omitempty" yaml:"timeout,omitempty" validate:"duration"`
	Var         string            `json:"var,omitempty" yaml:"var,omitempty" validate:"max=64"`
	If          string            `json:"if,omitempty" yaml:"if,omitempty" validate:"expr"`
	Parallel    *Parallel         `json:"parallel,omitempty" yaml:"parallel,omitempty"`
	Each        *Each             `json:"each,omitempty" yaml:"each,omitempty"`
	SubJob      *SubJob           `json:"subjob,omitempty" yaml:"subjob,omitempty"`
	GPUs        string            `json:"gpus,omitempty" yaml:"gpus,omitempty"`
	Tags        []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Workdir     string            `json:"workdir,omitempty" yaml:"workdir,omitempty" validate:"max=256"`
	Priority    int               `json:"priority,omitempty" yaml:"priority,omitempty" validate:"min=0,max=9"`
}

type SubJob struct {
	ID          string            `json:"id,omitempty"`
	Name        string            `json:"name,omitempty" yaml:"name,omitempty" validate:"required"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Tasks       []Task            `json:"tasks,omitempty" yaml:"tasks,omitempty" validate:"required"`
	Inputs      map[string]string `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	Secrets     map[string]string `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	AutoDelete  *AutoDelete       `json:"autoDelete,omitempty" yaml:"autoDelete,omitempty"`
	Output      string            `json:"output,omitempty" yaml:"output,omitempty"`
	Detached    bool              `json:"detached,omitempty" yaml:"detached,omitempty"`
	Webhooks    []Webhook         `json:"webhooks,omitempty" yaml:"webhooks,omitempty" validate:"dive"`
}

type Each struct {
	Var         string `json:"var,omitempty" yaml:"var,omitempty" `
	List        string `json:"list,omitempty" yaml:"list,omitempty" validate:"required,expr"`
	Task        Task   `json:"task,omitempty" yaml:"task,omitempty" validate:"required"`
	Concurrency int    `json:"concurrency,omitempty" yaml:"concurrency,omitempty" validate:"min=0,max=99999"`
}

type Parallel struct {
	Tasks []Task `json:"tasks,omitempty" yaml:"tasks,omitempty" validate:"required,min=1,dive"`
}

type Retry struct {
	Limit int `json:"limit,omitempty" yaml:"limit,omitempty" validate:"required,min=1,max=10"`
}

type Limits struct {
	CPUs   string `json:"cpus,omitempty" yaml:"cpus,omitempty"`
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`
}

type Registry struct {
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
}

type Mount struct {
	Type   string `json:"type,omitempty" yaml:"type,omitempty"`
	Source string `json:"source,omitempty" yaml:"source,omitempty"`
	Target string `json:"target,omitempty" yaml:"target,omitempty"`
}

type AuxTask struct {
	Name        string            `json:"name,omitempty" yaml:"name,omitempty" validate:"required"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	CMD         []string          `json:"cmd,omitempty" yaml:"cmd,omitempty"`
	Entrypoint  []string          `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	Run         string            `json:"run,omitempty" yaml:"run,omitempty"`
	Image       string            `json:"image,omitempty" yaml:"image,omitempty"`
	Registry    *Registry         `json:"registry,omitempty" yaml:"registry,omitempty"`
	Env         map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Files       map[string]string `json:"files,omitempty" yaml:"files,omitempty"`
	Timeout     string            `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

type SidecarTask struct {
	Name        string            `json:"name,omitempty" yaml:"name,omitempty" validate:"required"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	CMD         []string          `json:"cmd,omitempty" yaml:"cmd,omitempty"`
	Entrypoint  []string          `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	Run         string            `json:"run,omitempty" yaml:"run,omitempty"`
	Image       string            `json:"image,omitempty" yaml:"image,omitempty"`
	Registry    *Registry         `json:"registry,omitempty" yaml:"registry,omitempty"`
	Env         map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Files       map[string]string `json:"files,omitempty" yaml:"files,omitempty"`
	Timeout     string            `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Probe       *Probe            `json:"probe,omitempty" yaml:"probe,omitempty"`
}

type Probe struct {
	Path    string `json:"path,omitempty" yaml:"path,omitempty" validate:"required,max=256"`
	Port    int    `json:"port,omitempty" yaml:"port,omitempty" validate:"required,min=1,max=65535"`
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty" validate:"duration"`
}

func (m Mount) toMount() tork.Mount {
	return tork.Mount{
		Type:   m.Type,
		Source: m.Source,
		Target: m.Target,
	}
}

func (i AuxTask) toTask() *tork.Task {
	var registry *tork.Registry
	if i.Registry != nil {
		registry = &tork.Registry{
			Username: i.Registry.Username,
			Password: i.Registry.Password,
		}
	}
	return &tork.Task{
		Name:        i.Name,
		Description: i.Description,
		CMD:         i.CMD,
		Entrypoint:  i.Entrypoint,
		Run:         i.Run,
		Image:       i.Image,
		Env:         i.Env,
		Timeout:     i.Timeout,
		Files:       i.Files,
		Registry:    registry,
	}
}

func (i SidecarTask) toTask() *tork.Task {
	var registry *tork.Registry
	if i.Registry != nil {
		registry = &tork.Registry{
			Username: i.Registry.Username,
			Password: i.Registry.Password,
		}
	}
	var probe *tork.Probe
	if i.Probe != nil {
		probe = &tork.Probe{
			Path:    i.Probe.Path,
			Port:    i.Probe.Port,
			Timeout: i.Probe.Timeout,
		}
	}
	return &tork.Task{
		Name:        i.Name,
		Description: i.Description,
		CMD:         i.CMD,
		Entrypoint:  i.Entrypoint,
		Run:         i.Run,
		Image:       i.Image,
		Env:         i.Env,
		Timeout:     i.Timeout,
		Files:       i.Files,
		Registry:    registry,
		Probe:       probe,
	}
}

func (i Task) toTask() *tork.Task {
	pre := toAuxTasks(i.Pre)
	post := toAuxTasks(i.Post)
	sidecars := toSidecarTasks(i.Sidecars)
	var retry *tork.TaskRetry
	if i.Retry != nil {
		retry = i.Retry.toTaskRetry()
	}
	var limits *tork.TaskLimits
	if i.Limits != nil {
		limits = i.Limits.toTaskLimits()
	}
	var each *tork.EachTask
	if i.Each != nil {
		each = &tork.EachTask{
			Var:         i.Each.Var,
			List:        i.Each.List,
			Task:        i.Each.Task.toTask(),
			Concurrency: i.Each.Concurrency,
		}
	}
	var subjob *tork.SubJobTask
	if i.SubJob != nil {
		webhooks := make([]*tork.Webhook, len(i.SubJob.Webhooks))
		for i, wh := range i.SubJob.Webhooks {
			webhooks[i] = wh.toWebhook()
		}
		subjob = &tork.SubJobTask{
			Name:        i.SubJob.Name,
			Description: i.SubJob.Description,
			Tasks:       toTasks(i.SubJob.Tasks),
			Inputs:      maps.Clone(i.SubJob.Inputs),
			Secrets:     maps.Clone(i.SubJob.Secrets),
			Output:      i.SubJob.Output,
			Detached:    i.SubJob.Detached,
			Webhooks:    webhooks,
		}
		if i.SubJob.AutoDelete != nil {
			subjob.AutoDelete = &tork.AutoDelete{
				After: i.SubJob.AutoDelete.After,
			}
		}
	}
	var parallel *tork.ParallelTask
	if i.Parallel != nil {
		parallel = &tork.ParallelTask{
			Tasks: toTasks(i.Parallel.Tasks),
		}
	}
	var registry *tork.Registry
	if i.Registry != nil {
		registry = &tork.Registry{
			Username: i.Registry.Username,
			Password: i.Registry.Password,
		}
	}
	return &tork.Task{
		Name:        i.Name,
		Description: i.Description,
		CMD:         i.CMD,
		Entrypoint:  i.Entrypoint,
		Run:         i.Run,
		Image:       i.Image,
		Registry:    registry,
		Env:         i.Env,
		Files:       i.Files,
		Queue:       i.Queue,
		Pre:         pre,
		Post:        post,
		Sidecars:    sidecars,
		Mounts:      toMounts(i.Mounts),
		Networks:    i.Networks,
		Retry:       retry,
		Limits:      limits,
		Timeout:     i.Timeout,
		Var:         i.Var,
		If:          i.If,
		Parallel:    parallel,
		Each:        each,
		SubJob:      subjob,
		GPUs:        i.GPUs,
		Tags:        i.Tags,
		Workdir:     i.Workdir,
		Priority:    i.Priority,
	}
}

func toMounts(ms []Mount) []tork.Mount {
	result := make([]tork.Mount, len(ms))
	for i, m := range ms {
		result[i] = m.toMount()
	}
	return result
}

func toAuxTasks(tis []AuxTask) []*tork.Task {
	result := make([]*tork.Task, len(tis))
	for i, ti := range tis {
		result[i] = ti.toTask()
	}
	return result
}

func toSidecarTasks(tis []SidecarTask) []*tork.Task {
	result := make([]*tork.Task, len(tis))
	for i, ti := range tis {
		result[i] = ti.toTask()
	}
	return result
}
func toTasks(tis []Task) []*tork.Task {
	result := make([]*tork.Task, len(tis))
	for i, ti := range tis {
		result[i] = ti.toTask()
	}
	return result
}

func (l *Limits) toTaskLimits() *tork.TaskLimits {
	return &tork.TaskLimits{
		CPUs:   l.CPUs,
		Memory: l.Memory,
	}
}

func (r *Retry) toTaskRetry() *tork.TaskRetry {
	return &tork.TaskRetry{
		Limit: r.Limit,
	}
}

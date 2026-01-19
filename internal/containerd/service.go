package containerd

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	tasksv1 "github.com/containerd/containerd/api/services/tasks/v1"
	tasktypes "github.com/containerd/containerd/api/types/task"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/defaults"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/mount"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrTaskStopTimeout = errors.New("timeout waiting for task to stop")
)

type PullImageOptions struct {
	Unpack      bool
	Snapshotter string
}

type DeleteImageOptions struct {
	Synchronous bool
}

type CreateContainerRequest struct {
	ID          string
	ImageRef    string
	SnapshotID  string
	Snapshotter string
	Labels      map[string]string
	SpecOpts    []oci.SpecOpts
}

type DeleteContainerOptions struct {
	CleanupSnapshot bool
}

type StartTaskOptions struct {
	UseStdio bool
	Terminal bool
	FIFODir  string
}

type StopTaskOptions struct {
	Signal  syscall.Signal
	Timeout time.Duration
	Force   bool
}

type DeleteTaskOptions struct {
	Force bool
}

type ExecTaskRequest struct {
	Args     []string
	Env      []string
	WorkDir  string
	Terminal bool
	UseStdio bool
}

type ExecTaskResult struct {
	ExitCode uint32
}

type SnapshotCommitResult struct {
	VersionSnapshotID string
	ActiveSnapshotID  string
}

type ListTasksOptions struct {
	Filter string
}

type TaskInfo struct {
	ContainerID string
	ID          string
	PID         uint32
	Status      tasktypes.Status
	ExitStatus  uint32
}

type Service interface {
	PullImage(ctx context.Context, ref string, opts *PullImageOptions) (containerd.Image, error)
	GetImage(ctx context.Context, ref string) (containerd.Image, error)
	ListImages(ctx context.Context) ([]containerd.Image, error)
	DeleteImage(ctx context.Context, ref string, opts *DeleteImageOptions) error

	CreateContainer(ctx context.Context, req CreateContainerRequest) (containerd.Container, error)
	GetContainer(ctx context.Context, id string) (containerd.Container, error)
	ListContainers(ctx context.Context) ([]containerd.Container, error)
	DeleteContainer(ctx context.Context, id string, opts *DeleteContainerOptions) error

	StartTask(ctx context.Context, containerID string, opts *StartTaskOptions) (containerd.Task, error)
	GetTask(ctx context.Context, containerID string) (containerd.Task, error)
	ListTasks(ctx context.Context, opts *ListTasksOptions) ([]TaskInfo, error)
	StopTask(ctx context.Context, containerID string, opts *StopTaskOptions) error
	DeleteTask(ctx context.Context, containerID string, opts *DeleteTaskOptions) error
	ExecTask(ctx context.Context, containerID string, req ExecTaskRequest) (ExecTaskResult, error)
	ListContainersByLabel(ctx context.Context, key, value string) ([]containerd.Container, error)
	CommitSnapshot(ctx context.Context, snapshotter, name, key string) error
	PrepareSnapshot(ctx context.Context, snapshotter, key, parent string) error
	CreateContainerFromSnapshot(ctx context.Context, req CreateContainerRequest) (containerd.Container, error)
	SnapshotMounts(ctx context.Context, snapshotter, key string) ([]mount.Mount, error)
}

type DefaultService struct {
	client    *containerd.Client
	namespace string
}

func NewDefaultService(client *containerd.Client, namespace string) *DefaultService {
	if namespace == "" {
		namespace = DefaultNamespace
	}
	return &DefaultService{
		client:    client,
		namespace: namespace,
	}
}

func (s *DefaultService) PullImage(ctx context.Context, ref string, opts *PullImageOptions) (containerd.Image, error) {
	if ref == "" {
		return nil, ErrInvalidArgument
	}

	ctx = s.withNamespace(ctx)
	pullOpts := []containerd.RemoteOpt{}
	if opts == nil || opts.Unpack {
		pullOpts = append(pullOpts, containerd.WithPullUnpack)
	}
	if opts != nil && opts.Snapshotter != "" {
		pullOpts = append(pullOpts, containerd.WithPullSnapshotter(opts.Snapshotter))
	}

	return s.client.Pull(ctx, ref, pullOpts...)
}

func (s *DefaultService) GetImage(ctx context.Context, ref string) (containerd.Image, error) {
	if ref == "" {
		return nil, ErrInvalidArgument
	}
	ctx = s.withNamespace(ctx)
	return s.client.GetImage(ctx, ref)
}

func (s *DefaultService) ListImages(ctx context.Context) ([]containerd.Image, error) {
	ctx = s.withNamespace(ctx)
	return s.client.ListImages(ctx)
}

func (s *DefaultService) DeleteImage(ctx context.Context, ref string, opts *DeleteImageOptions) error {
	if ref == "" {
		return ErrInvalidArgument
	}
	ctx = s.withNamespace(ctx)
	deleteOpts := []images.DeleteOpt{}
	if opts != nil && opts.Synchronous {
		deleteOpts = append(deleteOpts, images.SynchronousDelete())
	}
	return s.client.ImageService().Delete(ctx, ref, deleteOpts...)
}

func (s *DefaultService) CreateContainer(ctx context.Context, req CreateContainerRequest) (containerd.Container, error) {
	if req.ID == "" || req.ImageRef == "" {
		return nil, ErrInvalidArgument
	}

	ctx = s.withNamespace(ctx)
	pullOpts := &PullImageOptions{
		Unpack:      true,
		Snapshotter: req.Snapshotter,
	}
	image, err := s.PullImage(ctx, req.ImageRef, pullOpts)
	if err != nil {
		return nil, err
	}

	snapshotID := req.SnapshotID
	if snapshotID == "" {
		snapshotID = req.ID
	}

	specOpts := []oci.SpecOpts{
		oci.WithDefaultSpecForPlatform("linux/" + runtime.GOARCH),
		oci.WithImageConfig(image),
	}
	if len(req.SpecOpts) > 0 {
		specOpts = append(specOpts, req.SpecOpts...)
	}

	containerOpts := []containerd.NewContainerOpts{
		containerd.WithImage(image),
		containerd.WithNewSnapshot(snapshotID, image),
		containerd.WithNewSpec(specOpts...),
	}
	runtimeName := s.client.Runtime()
	if runtimeName == "" {
		runtimeName = defaults.DefaultRuntime
		if runtimeName == "" {
			runtimeName = "io.containerd.runc.v2"
		}
	}
	containerOpts = append(containerOpts, containerd.WithRuntime(runtimeName, nil))
	if req.Snapshotter != "" {
		containerOpts = append(containerOpts, containerd.WithSnapshotter(req.Snapshotter))
	}
	if len(req.Labels) > 0 {
		containerOpts = append(containerOpts, containerd.WithContainerLabels(req.Labels))
	}

	return s.client.NewContainer(ctx, req.ID, containerOpts...)
}

func (s *DefaultService) GetContainer(ctx context.Context, id string) (containerd.Container, error) {
	if id == "" {
		return nil, ErrInvalidArgument
	}
	ctx = s.withNamespace(ctx)
	return s.client.LoadContainer(ctx, id)
}

func (s *DefaultService) ListContainers(ctx context.Context) ([]containerd.Container, error) {
	ctx = s.withNamespace(ctx)
	return s.client.Containers(ctx)
}

func (s *DefaultService) DeleteContainer(ctx context.Context, id string, opts *DeleteContainerOptions) error {
	if id == "" {
		return ErrInvalidArgument
	}

	ctx = s.withNamespace(ctx)
	container, err := s.client.LoadContainer(ctx, id)
	if err != nil {
		return err
	}

	deleteOpts := []containerd.DeleteOpts{}
	cleanupSnapshot := true
	if opts != nil {
		cleanupSnapshot = opts.CleanupSnapshot
	}
	if cleanupSnapshot {
		deleteOpts = append(deleteOpts, containerd.WithSnapshotCleanup)
	}

	return container.Delete(ctx, deleteOpts...)
}

func (s *DefaultService) StartTask(ctx context.Context, containerID string, opts *StartTaskOptions) (containerd.Task, error) {
	if containerID == "" {
		return nil, ErrInvalidArgument
	}

	ctx = s.withNamespace(ctx)
	container, err := s.client.LoadContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	var cioOpts []cio.Opt
	if opts == nil || opts.UseStdio {
		cioOpts = append(cioOpts, cio.WithStdio)
	}
	if opts != nil && opts.Terminal {
		cioOpts = append(cioOpts, cio.WithTerminal)
	}
	if opts != nil && opts.FIFODir != "" {
		cioOpts = append(cioOpts, cio.WithFIFODir(opts.FIFODir))
	}
	ioCreator := cio.NewCreator(cioOpts...)

	task, err := container.NewTask(ctx, ioCreator)
	if err != nil {
		return nil, err
	}
	if err := task.Start(ctx); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *DefaultService) GetTask(ctx context.Context, containerID string) (containerd.Task, error) {
	if containerID == "" {
		return nil, ErrInvalidArgument
	}

	ctx = s.withNamespace(ctx)
	container, err := s.client.LoadContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}
	return container.Task(ctx, nil)
}

func (s *DefaultService) ListTasks(ctx context.Context, opts *ListTasksOptions) ([]TaskInfo, error) {
	ctx = s.withNamespace(ctx)
	request := &tasksv1.ListTasksRequest{}
	if opts != nil {
		request.Filter = opts.Filter
	}

	response, err := s.client.TaskService().List(ctx, request)
	if err != nil {
		return nil, err
	}

	tasks := make([]TaskInfo, 0, len(response.Tasks))
	for _, task := range response.Tasks {
		tasks = append(tasks, TaskInfo{
			ContainerID: task.ContainerID,
			ID:          task.ID,
			PID:         task.Pid,
			Status:      task.Status,
			ExitStatus:  task.ExitStatus,
		})
	}

	return tasks, nil
}

func (s *DefaultService) StopTask(ctx context.Context, containerID string, opts *StopTaskOptions) error {
	if containerID == "" {
		return ErrInvalidArgument
	}

	ctx = s.withNamespace(ctx)
	task, err := s.GetTask(ctx, containerID)
	if err != nil {
		return err
	}

	signal := syscall.SIGTERM
	timeout := 10 * time.Second
	force := false
	if opts != nil {
		if opts.Signal != 0 {
			signal = opts.Signal
		}
		if opts.Timeout != 0 {
			timeout = opts.Timeout
		}
		force = opts.Force
	}

	if err := task.Kill(ctx, signal); err != nil {
		return err
	}

	statusC, err := task.Wait(ctx)
	if err != nil {
		return err
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-statusC:
		return nil
	case <-timer.C:
		if force {
			if err := task.Kill(ctx, syscall.SIGKILL); err != nil {
				return fmt.Errorf("force kill failed: %w", err)
			}
			<-statusC
			return nil
		}
		return ErrTaskStopTimeout
	}
}

func (s *DefaultService) DeleteTask(ctx context.Context, containerID string, opts *DeleteTaskOptions) error {
	if containerID == "" {
		return ErrInvalidArgument
	}

	ctx = s.withNamespace(ctx)
	task, err := s.GetTask(ctx, containerID)
	if err != nil {
		return err
	}

	if opts != nil && opts.Force {
		_ = task.Kill(ctx, syscall.SIGKILL)
	}

	_, err = task.Delete(ctx)
	return err
}

func (s *DefaultService) ExecTask(ctx context.Context, containerID string, req ExecTaskRequest) (ExecTaskResult, error) {
	if containerID == "" || len(req.Args) == 0 {
		return ExecTaskResult{}, ErrInvalidArgument
	}

	ctx = s.withNamespace(ctx)
	container, err := s.client.LoadContainer(ctx, containerID)
	if err != nil {
		return ExecTaskResult{}, err
	}

	spec, err := container.Spec(ctx)
	if err != nil {
		return ExecTaskResult{}, err
	}
	if spec.Process == nil {
		spec.Process = &specs.Process{}
	}

	if len(req.Env) > 0 {
		if err := oci.WithEnv(req.Env)(ctx, nil, nil, spec); err != nil {
			return ExecTaskResult{}, err
		}
	}

	spec.Process.Args = req.Args
	if req.WorkDir != "" {
		spec.Process.Cwd = req.WorkDir
	}
	if req.Terminal {
		spec.Process.Terminal = true
	}

	task, err := container.Task(ctx, nil)
	if err != nil {
		return ExecTaskResult{}, err
	}

	ioOpts := []cio.Opt{}
	if req.UseStdio {
		ioOpts = append(ioOpts, cio.WithStdio)
	}
	if req.Terminal {
		ioOpts = append(ioOpts, cio.WithTerminal)
	}
	ioCreator := cio.NewCreator(ioOpts...)

	execID := fmt.Sprintf("exec-%d", time.Now().UnixNano())
	process, err := task.Exec(ctx, execID, spec.Process, ioCreator)
	if err != nil {
		return ExecTaskResult{}, err
	}
	defer process.Delete(ctx)

	statusC, err := process.Wait(ctx)
	if err != nil {
		return ExecTaskResult{}, err
	}
	if err := process.Start(ctx); err != nil {
		return ExecTaskResult{}, err
	}

	status := <-statusC
	code, _, err := status.Result()
	if err != nil {
		return ExecTaskResult{}, err
	}

	return ExecTaskResult{ExitCode: code}, nil
}

func (s *DefaultService) ListContainersByLabel(ctx context.Context, key, value string) ([]containerd.Container, error) {
	if key == "" {
		return nil, ErrInvalidArgument
	}

	ctx = s.withNamespace(ctx)
	containers, err := s.client.Containers(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]containerd.Container, 0, len(containers))
	for _, container := range containers {
		info, err := container.Info(ctx)
		if err != nil {
			return nil, err
		}
		if labelValue, ok := info.Labels[key]; ok && (value == "" || value == labelValue) {
			filtered = append(filtered, container)
		}
	}
	return filtered, nil
}

func (s *DefaultService) CommitSnapshot(ctx context.Context, snapshotter, name, key string) error {
	if snapshotter == "" || name == "" || key == "" {
		return ErrInvalidArgument
	}
	ctx = s.withNamespace(ctx)
	return s.client.SnapshotService(snapshotter).Commit(ctx, name, key)
}

func (s *DefaultService) PrepareSnapshot(ctx context.Context, snapshotter, key, parent string) error {
	if snapshotter == "" || key == "" || parent == "" {
		return ErrInvalidArgument
	}
	ctx = s.withNamespace(ctx)
	_, err := s.client.SnapshotService(snapshotter).Prepare(ctx, key, parent)
	return err
}

func (s *DefaultService) CreateContainerFromSnapshot(ctx context.Context, req CreateContainerRequest) (containerd.Container, error) {
	if req.ID == "" || req.SnapshotID == "" {
		return nil, ErrInvalidArgument
	}

	ctx = s.withNamespace(ctx)

	imageRef := req.ImageRef
	if imageRef == "" {
		return nil, ErrInvalidArgument
	}

	image, err := s.GetImage(ctx, imageRef)
	if err != nil {
		image, err = s.PullImage(ctx, imageRef, &PullImageOptions{
			Unpack:      true,
			Snapshotter: req.Snapshotter,
		})
		if err != nil {
			return nil, err
		}
	}

	specOpts := []oci.SpecOpts{
		oci.WithDefaultSpecForPlatform("linux/" + runtime.GOARCH),
		oci.WithImageConfig(image),
	}
	if len(req.SpecOpts) > 0 {
		specOpts = append(specOpts, req.SpecOpts...)
	}

	containerOpts := []containerd.NewContainerOpts{
		containerd.WithImage(image),
		containerd.WithSnapshot(req.SnapshotID),
		containerd.WithNewSpec(specOpts...),
	}
	if req.Snapshotter != "" {
		containerOpts = append(containerOpts, containerd.WithSnapshotter(req.Snapshotter))
	}
	if len(req.Labels) > 0 {
		containerOpts = append(containerOpts, containerd.WithContainerLabels(req.Labels))
	}

	runtimeName := s.client.Runtime()
	if runtimeName == "" {
		runtimeName = defaults.DefaultRuntime
		if runtimeName == "" {
			runtimeName = "io.containerd.runc.v2"
		}
	}
	containerOpts = append(containerOpts, containerd.WithRuntime(runtimeName, nil))

	return s.client.NewContainer(ctx, req.ID, containerOpts...)
}

func (s *DefaultService) SnapshotMounts(ctx context.Context, snapshotter, key string) ([]mount.Mount, error) {
	if snapshotter == "" || key == "" {
		return nil, ErrInvalidArgument
	}
	ctx = s.withNamespace(ctx)
	return s.client.SnapshotService(snapshotter).Mounts(ctx, key)
}

func (s *DefaultService) withNamespace(ctx context.Context) context.Context {
	return namespaces.WithNamespace(ctx, s.namespace)
}

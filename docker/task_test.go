package docker_test

import (
	"testing"

	"shanhu.io/std/docker"
	"shanhu.io/std/docker/dockertest"
	"shanhu.io/std/errcode"
)

func TestRunTask(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc"})
		if err := docker.RunTask(docker.NewCont(d.Client, "abc"), "echo hello"); err != nil {
			t.Errorf("RunTask: %v", err)
		}
	})

	t.Run("non-zero exit", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc"})
		d.SetExecResponse(dockertest.ExecResponse{ExitCode: 2})
		err := docker.RunTask(docker.NewCont(d.Client, "abc"), "false")
		if err == nil {
			t.Fatal("RunTask: expected error on non-zero exit")
		}
		if err.Error() != "exit value: 2" {
			t.Errorf("error: got %q, want %q", err.Error(), "exit value: 2")
		}
	})

	t.Run("exec error", func(t *testing.T) {
		d := newDaemon(t)
		err := docker.RunTask(docker.NewCont(d.Client, "missing"), "echo hi")
		if err == nil {
			t.Fatal("RunTask: expected error on missing container")
		}
		if !errcode.IsNotFound(err) {
			t.Errorf("expected NotFound, got %v", err)
		}
	})
}

func TestRunTasks(t *testing.T) {
	t.Run("all succeed", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc"})
		if err := docker.RunTasks(docker.NewCont(d.Client, "abc"), []string{
			"echo one",
			"echo two",
			"echo three",
		}); err != nil {
			t.Errorf("RunTasks: %v", err)
		}
	})

	t.Run("empty", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc"})
		if err := docker.RunTasks(docker.NewCont(d.Client, "abc"), nil); err != nil {
			t.Errorf("RunTasks(nil): %v", err)
		}
	})

	t.Run("stops on first failure", func(t *testing.T) {
		d := newDaemon(t)
		d.AddContainer(&dockertest.Container{ID: "abc"})
		d.SetExecResponse(dockertest.ExecResponse{ExitCode: 1})
		err := docker.RunTasks(docker.NewCont(d.Client, "abc"), []string{
			"failing-cmd",
			"never-runs",
		})
		if err == nil {
			t.Fatal("RunTasks: expected error")
		}
	})
}

package main

import (
"time"
"syscall"
	"fmt"
	"path/filepath"
	"os"
	"os/exec"
)

type App struct {
	Name string
	Dir string
	Port int
	*os.Process
}

func StartApp(name, bin string) (*App, error) {
	app := &App {
		Name: name,
		Dir: bin,
	}
	cmd := exec.Command(filepath.Join(bin, "main"))
	copy(cmd.Env, os.Environ())
	app.Port = findFreePort()
	cmd.Env = append(cmd.Env, fmt.Sprintf("PORT=%d", app.Port))
	e := cmd.Start()
	if e != nil {
		return nil, e
	}
	app.Process = cmd.Process
	return app, nil
}

func (a *App) Destroy() error {
	e := a.Process.Signal(syscall.SIGTERM)
	if e != nil {
		return e
	}
	t := time.AfterFunc(5*time.Second, func() {
		a.Process.Kill()
	})
	_, e = a.Process.Wait()
	if e != nil {
		return e
	}
	t.Stop()
	e = os.RemoveAll(a.Dir)
	if e != nil {
		return e
	}
	return nil
}




package runtime

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Process struct {
	Cmd    *exec.Cmd
	Cancel context.CancelFunc
	Done   chan Result
}

type Runner struct {
	Bin string
	Dir string
}

func (r Runner) Start(ctx context.Context, args []string) (*Process, error) {
	cctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(cctx, r.Bin, args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	cmd.SysProcAttr = sysProcAttr()
	var outBuf, errBuf bytes.Buffer
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}
	p := &Process{Cmd: cmd, Cancel: cancel, Done: make(chan Result, 1)}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); io.Copy(&outBuf, stdout) }()
	go func() { defer wg.Done(); io.Copy(&errBuf, stderr) }()
	go func() {
		werr := cmd.Wait()
		wg.Wait()
		code := 0
		if werr != nil {
			if ee, ok := werr.(*exec.ExitError); ok {
				code = ee.ExitCode()
			} else {
				code = -1
			}
		}
		p.Done <- Result{Stdout: outBuf.String(), Stderr: errBuf.String(), ExitCode: code}
	}()
	return p, nil
}

func (r Runner) Run(ctx context.Context, timeout time.Duration, args []string) (Result, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	p, err := r.Start(cctx, args)
	if err != nil {
		return Result{}, err
	}
	select {
	case res := <-p.Done:
		return res, nil
	case <-cctx.Done():
		KillProcessTree(p.Cmd)
		<-p.Done
		return Result{}, fmt.Errorf("command timed out or cancelled: %w", cctx.Err())
	}
}

package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Manager struct {
	claudeBinary string
	workingDir   string
	timeout      time.Duration
}

func NewManager(claudeBinary, workingDir string, timeout time.Duration) *Manager {
	return &Manager{
		claudeBinary: claudeBinary,
		workingDir:   workingDir,
		timeout:      timeout,
	}
}

func (m *Manager) ExecuteTask(ctx context.Context, task string) error {
	log.Printf("Starting Claude Code agent with task: %s", task)

	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, m.claudeBinary, "--dangerously-skip-permissions", task)
	cmd.Dir = m.workingDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start claude process: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		m.streamOutput(stdout, "STDOUT")
	}()

	go func() {
		defer wg.Done()
		m.streamOutput(stderr, "STDERR")
	}()

	err = cmd.Wait()

	// Wait for all output to be read
	wg.Wait()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("task execution timed out after %v", m.timeout)
		}
		log.Printf("Claude process exited with error: %v", err)
		return fmt.Errorf("claude process failed: %w", err)
	}

	log.Printf("Task completed successfully")
	return nil
}

func (m *Manager) streamOutput(reader io.Reader, prefix string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			log.Printf("[%s] %s", prefix, line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading %s: %v", prefix, err)
	}
}

func (m *Manager) ValidateClaudeBinary() error {
	if _, err := os.Stat(m.claudeBinary); err != nil {
		if os.IsNotExist(err) {
			path, err := exec.LookPath("claude")
			if err != nil {
				return fmt.Errorf("claude binary not found in PATH or at specified location")
			}
			m.claudeBinary = path
		} else {
			return fmt.Errorf("error checking claude binary: %w", err)
		}
	}
	return nil
}

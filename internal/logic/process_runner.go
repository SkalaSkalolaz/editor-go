package logic

import (
	"context"
	"os/exec"
	"sync"
)

// ProcessRunner manages background processes (like go run) with cancellation.
type ProcessRunner struct {
	mu         sync.Mutex
	cancelFunc context.CancelFunc
	isRunning  bool
}

func NewProcessRunner() *ProcessRunner {
	return &ProcessRunner{}
}

// StartCommand executes a command and streams output via callback.
// Returns a cancel function to stop it manually if needed.
func (pr *ProcessRunner) StartCommand(dir string, name string, args []string, onOutput func(string)) (<-chan error, func()) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	// If already running, cancel previous
	if pr.isRunning && pr.cancelFunc != nil {
		pr.cancelFunc()
	}

	ctx, cancel := context.WithCancel(context.Background())
	pr.cancelFunc = cancel
	pr.isRunning = true

	// Channel to signal completion
	done := make(chan error, 1)

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	// Используем кастомный Writer для перехвата вывода в реальном времени
	writer := &commandWriter{callback: onOutput}
	cmd.Stdout = writer
	cmd.Stderr = writer

	// Запускаем процесс асинхронно
	err := cmd.Start()
	if err != nil {
		pr.isRunning = false
		done <- err
		close(done)
		return done, func() {}
	}

	// Ждем завершения в горутине
	go func() {
		waitErr := cmd.Wait()

		pr.mu.Lock()
		pr.isRunning = false
		pr.cancelFunc = nil
		pr.mu.Unlock()

		done <- waitErr
		close(done)
	}()

	userCancel := func() {
		cancel()
	}

	return done, userCancel
}

// Вспомогательная структура для трансляции вывода
type commandWriter struct {
	callback func(string)
}

func (cw *commandWriter) Write(p []byte) (n int, err error) {
	if cw.callback != nil {
		cw.callback(string(p))
	}
	return len(p), nil
}

func (pr *ProcessRunner) IsRunning() bool {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return pr.isRunning
}

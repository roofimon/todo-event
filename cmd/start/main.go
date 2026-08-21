package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var serviceNames = []string{"welcome", "credit", "audit", "api"}

type runningService struct {
	name string
	cmd  *exec.Cmd
	done <-chan struct{}
}

type serviceExit struct {
	name string
	err  error
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "start:", err)
		os.Exit(1)
	}
}

func run() error {
	repoDir, err := findRepoDir()
	if err != nil {
		return err
	}
	if err := loadEnv(filepath.Join(repoDir, ".env")); err != nil {
		return err
	}
	for _, command := range []string{"docker", "go"} {
		if _, err := exec.LookPath(command); err != nil {
			return fmt.Errorf("required command not found: %s", command)
		}
	}

	fmt.Println("Starting MongoDB, NATS, Loki, and Grafana...")
	if err := runCommand(repoDir, "docker", "compose", "-f", "compose.yml", "up", "-d", "--wait"); err != nil {
		return fmt.Errorf("start infrastructure: %w", err)
	}

	buildDir, err := os.MkdirTemp("", "todoe-services-")
	if err != nil {
		return fmt.Errorf("create build directory: %w", err)
	}
	defer os.RemoveAll(buildDir)

	fmt.Println("Building Go services...")
	for _, name := range serviceNames {
		output := filepath.Join(buildDir, name)
		if err := runCommand(repoDir, "go", "build", "-o", output, "./cmd/"+name); err != nil {
			return fmt.Errorf("build %s: %w", name, err)
		}
	}

	exits := make(chan serviceExit, len(serviceNames))
	services := make([]runningService, 0, len(serviceNames))
	defer func() { stopServices(services) }()

	// Subscribers start first so they are ready before the API publishes events.
	for _, name := range serviceNames {
		service, err := startService(repoDir, filepath.Join(buildDir, name), name, exits)
		if err != nil {
			return err
		}
		services = append(services, service)

		select {
		case exited := <-exits:
			return fmt.Errorf("%s exited during startup: %w", exited.name, exited.err)
		case <-time.After(time.Second):
		}
	}

	fmt.Println()
	fmt.Println("All services are running.")
	fmt.Println("API:     http://localhost:3000")
	fmt.Println("Grafana: http://localhost:3001")
	fmt.Println("Press Ctrl+C to stop the Go services. Docker services remain running.")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case <-signals:
		return nil
	case exited := <-exits:
		if exited.err == nil {
			return fmt.Errorf("%s exited unexpectedly", exited.name)
		}
		return fmt.Errorf("%s exited unexpectedly: %w", exited.name, exited.err)
	}
}

func findRepoDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not find repository root containing go.mod")
		}
		dir = parent
	}
}

func loadEnv(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open .env: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return fmt.Errorf("parse .env line %d", lineNumber)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"')) {
			value = value[1 : len(value)-1]
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s from .env: %w", key, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read .env: %w", err)
	}
	return nil
}

func runCommand(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func startService(dir, binary, name string, exits chan<- serviceExit) (runningService, error) {
	fmt.Printf("Starting %s...\n", name)
	cmd := exec.Command(binary)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return runningService{}, fmt.Errorf("start %s: %w", name, err)
	}
	done := make(chan struct{})
	go func() {
		err := cmd.Wait()
		close(done)
		exits <- serviceExit{name: name, err: err}
	}()
	return runningService{name: name, cmd: cmd, done: done}, nil
}

func stopServices(services []runningService) {
	if len(services) == 0 {
		return
	}
	fmt.Println()
	fmt.Println("Stopping todoe services...")
	for _, service := range services {
		if service.cmd.Process != nil {
			_ = service.cmd.Process.Signal(os.Interrupt)
		}
	}

	deadline := time.Now().Add(3 * time.Second)
	for _, service := range services {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		select {
		case <-service.done:
		case <-time.After(remaining):
		}
	}
	for _, service := range services {
		select {
		case <-service.done:
			continue
		default:
		}
		if service.cmd.Process != nil {
			_ = service.cmd.Process.Kill()
		}
	}
}

package dockertest

// ExecResponse configures the output and exit code that the fake daemon
// returns for subsequent Cont.Exec calls. Stdout/Stderr are emitted as
// Docker's multiplexed log stream in start responses; ExitCode is reported
// from the exec inspect response.
type ExecResponse struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// execState tracks a single exec instance created via
// POST /containers/{id}/exec.
type execState struct {
	ID          string
	ContainerID string
	Cmd         []string
	Running     bool
	ExitCode    int
}

# Agent Orchestrator

A Go-based HTTP server that accepts task requests and orchestrates Claude Code agents to execute those tasks.

## Features

- RESTful HTTP API with POST endpoint for task submission
- Asynchronous task processing using Claude Code agents
- Configurable timeouts and working directories
- Request logging and error handling
- Graceful shutdown support

## Prerequisites

- Go 1.16 or higher
- Claude Code CLI (`claude` binary in PATH)

## Installation

```bash
go build -o agent-orchestrator .
```

## Usage

### Starting the Server

```bash
./agent-orchestrator [flags]
```

### Command-line Flags

- `-port`: Server port (default: 8080)
- `-kubeconfig`: Path to kubeconfig file (optional, defaults to standard locations)
- `-namespace`: Kubernetes namespace to launch agents in (default: "agents")
- `-task-timeout`: Timeout for task execution (default: 30m)

### Example

```bash
./agent-orchestrator -port 8080 -namespace agents -task-timeout 1h
```

## API Endpoints

### POST /api/tasks

Submit a task for Claude Code to execute. The orchestrator will spin up a Kubernetes Pod to run the agent.

**Request Body:**
```json
{
  "task": "implement a cool telegram bot"
}
```

**Response (202 Accepted):**
```json
{
  "status": "accepted",
  "message": "Task has been accepted and is being processed"
}
```

**Error Response (400 Bad Request):**
```json
{
  "error": "task field is required and cannot be empty"
}
```

### GET /health

Health check endpoint.

**Response (200 OK):**
```
OK
```

## Example Usage

### Using curl

```bash
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"task": "implement a cool telegram bot"}'
```

### Using httpie

```bash
http POST localhost:8080/api/tasks task="implement a cool telegram bot"
```

## Architecture

The project is structured as follows:

```
agent-orchestrator/
├── main.go                      # Main entry point
├── internal/
│   ├── server/
│   │   └── server.go           # HTTP server implementation
│   ├── handler/
│   │   ├── types.go            # Request/response types
│   │   └── task_handler.go    # Task endpoint handler
│   └── agent/
│       └── manager.go          # Claude Code process manager
├── go.mod
└── README.md
```

## How It Works

1. Client sends a POST request to `/api/tasks` with a task description
2. Server validates the request and returns 202 Accepted immediately
3. Task is executed asynchronously by creating a Kubernetes Pod
4. The Pod runs the Claude Code agent with `--dangerously-skip-permissions` flag and the task as argument
5. The orchestrator monitors the Pod status and cleans it up after completion or timeout
6. Process status and logs are available via Kubernetes tools (kubectl)

## Logging

All logs are written to stdout with timestamps:
- Request logging (method, path, duration)
- Task execution status
- Claude Code output (stdout/stderr)
- Error messages

## Security Considerations

- Input validation to prevent command injection
- Configurable timeouts to prevent resource exhaustion
- Process isolation using separate working directories
- Consider adding authentication for production use
- Consider adding rate limiting for production use

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o agent-orchestrator .
```

## License

MIT

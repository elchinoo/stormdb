# Signal Handling and Graceful Shutdown

stormdb now supports graceful shutdown when receiving SIGINT (Ctrl+C) or SIGTERM signals.

## Features

- **Graceful Shutdown**: When interrupted, stormdb will:
  - Stop accepting new work
  - Allow existing transactions to complete (up to 5 seconds)
  - Print a final summary with all collected metrics
  - Exit cleanly with appropriate status code

- **Signal Support**: 
  - `SIGINT` (Ctrl+C): Interactive interruption
  - `SIGTERM`: Programmatic termination (useful for containers/systemd)

## Usage

Start any workload and press Ctrl+C to see graceful shutdown:

```bash
# Start a workload
./stormdb -c config/config_pgv_1024.yaml

# Press Ctrl+C to interrupt
# stormdb will print:
# 🛑 Received signal interrupt, shutting down gracefully...
# 📊 Final Summary (interrupted):
# [metrics output]
```

## Implementation Details

- Uses Go's `os/signal` package for signal handling
- Creates a 5-second grace period for workload completion
- Enhanced metrics reporting shows interruption status
- All workloads respect context cancellation for clean shutdown

## Testing

Run the signal handling demo:

```bash
./demo_signal_handling.sh
```

Or test programmatically:

```bash
go test ./test/unit/signal_test.go -v
```

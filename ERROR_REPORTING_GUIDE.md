# API Error Reporting Enhancement

## Failed Test Debugging

When test runs fail (status: "failed"), the API now provides enhanced error information:

### Enhanced TestRun Response

Failed test runs now include:
- `error_message`: Primary error that caused the failure
- `error_details`: Additional context for debugging
- `logs_url`: Direct path to retrieve logs for troubleshooting

Example failed test response:
```json
{
  "id": 8,
  "status": "failed",
  "error_message": "Connection refused: could not connect to target database",
  "error_details": {
    "task_id": "test-8",
    "plugin": "tpcc-scalability", 
    "version": "1.0.0"
  },
  "logs_url": "/test-runs/8/logs",
  ...
}
```

### Logs Endpoint

Retrieve detailed execution logs for debugging:

**GET** `/test-runs/{id}/logs?limit=N`

Query parameters:
- `limit`: Number of log entries to return (default: 50, max: 1000)

Response includes:
- Log entries with timestamps, levels, and messages
- Total count of logs returned
- Useful for debugging plugin failures, connection issues, configuration problems

Example:
```bash
curl http://localhost:8080/test-runs/8/logs?limit=100
```

This provides complete execution context for troubleshooting failed test runs.

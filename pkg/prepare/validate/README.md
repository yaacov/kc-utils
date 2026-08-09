# validate -- input validation

Validates the required inputs before the prepare pipeline begins. It ensures that at least one disk is specified and that a mount-root directory is provided, creating the mount-root on disk if it does not already exist.

The single exported function performs two guards (non-zero disk count, non-empty mount root) and then calls `os.MkdirAll` to guarantee the mount-root directory tree exists. Any failure is returned as a descriptive error.

## Key exports

| Symbol | Role |
|--------|------|
| `Input` | Checks that disks and mount root are specified; creates the mount-root directory |

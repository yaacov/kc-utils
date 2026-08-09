# plugin -- generic plugin registry

Shared across all pluggable blocks in every pipeline stage. Lives in `common/` because the same `Registry[K,V]` type is instantiated by packages in prepare, convert-linux, convert-windows, and finalize — a single definition avoids duplicate locking logic and keeps the plugin contract uniform.

Provides a thread-safe, generic registry for mapping keys to implementation values. This is the foundation for the project's plugin architecture: converter blocks, UEFI editors, firstboot handlers, and registry editors all use `Registry` to register and look up their implementations at runtime.

`Registry[K, V]` wraps a `map[K]V` protected by a `sync.RWMutex`. Plugins call `Register` during `init()` to insert themselves under a key. At runtime the pipeline calls `Get` to retrieve a specific implementation, `All` to iterate over every registered plugin, or `List` to enumerate available keys. Read operations use a shared lock; writes take an exclusive lock.

## Key exports

| Symbol | Role |
|--------|------|
| `Registry[K, V]` | Generic, thread-safe map from keys to plugin implementations |
| `NewRegistry[K, V]()` | Create a new empty registry |
| `(*Registry).Register(key, impl)` | Add or replace an implementation under the given key |
| `(*Registry).Get(key)` | Look up an implementation by key; returns (value, ok) |
| `(*Registry).All()` | Return a snapshot copy of all registered implementations |
| `(*Registry).List()` | Return a slice of all registered keys |

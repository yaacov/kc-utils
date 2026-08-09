# converter -- converter binary selection

Selects which converter binary to run based on the OS inspection results of the guest. Different guest operating systems may require different conversion tools, and this package provides the abstraction point for that decision.

The `ConverterSelector` interface defines a single `Select` method that takes inspection data and returns the name of the converter to use. Implementations are registered in the global `Selectors` plugin registry, allowing the prepare pipeline to look up the appropriate selector at runtime without hard-coding converter logic.

## Key exports

| Symbol | Role |
|--------|------|
| `ConverterSelector` | Interface with `Select(inspect) (converterName, error)` method |
| `Selectors` | Global plugin registry of `ConverterSelector` implementations |

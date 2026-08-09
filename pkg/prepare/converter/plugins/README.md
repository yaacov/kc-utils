# converter plugins

`ConverterSelector` interface — map inspect results to `linux` or `windows` converter.

After kc-prepare has mounted and inspected the guest, the converter selector
decides which conversion binary to invoke next. The inspect data includes the
detected OS type (`linux` or `windows`), and the selector maps that to the
correct binary name. This is the final block in the prepare pipeline — its
output is written into the `PrepareOutput` JSON so the orchestrator knows
which converter to spawn.

| Key | Package | Description |
|-----|---------|-------------|
| `default` | default/ | OS family → converter binary name |

## default

**What it does:** Routes to `kc-convert-linux` for Linux guests and
`kc-convert-windows` for Windows guests.

**How it works:** Reads `inspect.Type` from the prepare output. Returns
`"kc-convert-linux"` for `"linux"`, `"kc-convert-windows"` for `"windows"`,
and an error for any unrecognized OS type (e.g. FreeBSD, which is not
currently supported).

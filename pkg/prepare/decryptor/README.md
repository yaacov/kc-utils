# decryptor -- disk decryption interface

Provides a pluggable interface for decrypting encrypted disk volumes (e.g., LUKS) during the prepare pipeline. When the guest has encrypted partitions, a registered decryptor is used to unlock them so that root discovery and mounting can proceed on the decrypted block devices.

The `Decryptor` interface defines two methods: `Decrypt`, which takes a device path and a key source and returns the path to the decrypted mapper device, and `Close`, which tears down the decrypted mapping. Implementations are registered in the global `Decryptors` plugin registry, keeping the core prepare logic independent of any specific encryption technology.

## Key exports

| Symbol | Role |
|--------|------|
| `Decryptor` | Interface with `Decrypt(device, keySource) (string, error)` and `Close() error` |
| `Decryptors` | Global plugin registry of `Decryptor` implementations |

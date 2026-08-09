# fstrim plugins

`Trimmer` interface — non-fatal filesystem trim before unmount.

After conversion the guest disk image may contain blocks that the guest OS has
freed but the image file still allocates. Trimming tells the underlying storage
layer to release those unused blocks, reducing the final disk image size and
improving I/O performance on thin-provisioned storage. The trim step runs after
all conversion work is complete but before the filesystems are unmounted, so the
guest filesystem is in a consistent state. Errors from trim are non-fatal — they
are logged but do not abort the finalize pipeline.

| Key | Package | Description |
|-----|---------|-------------|
| `default` | default/ | Trim unused blocks via the active guest handle |

## default

**What it does:** Performs filesystem TRIM (DISCARD) on each mounted guest
filesystem to release blocks that are no longer in use by the guest OS.

**How it works:** Retrieves the active `guest.Guest` handle (set by the
finalize pipeline during attach) and calls its `FSTrim` method on each
mountpoint. In direct-backend mode this runs the `fstrim` CLI tool against
the host mount; in guestfs mode the trim is performed via the libguestfs
appliance RPC. Returns an error if no active guest handle is available.

# File generation locks

`FS.ReadOrWrite` reads an existing file or calls a factory while holding a nonblocking advisory file lock. Its zero value is usable and concurrent calls are supported on Linux, macOS, FreeBSD, OpenBSD, NetBSD, and DragonFly BSD.

All cooperating writers must use the same path and leave the `.lock` file in place. Contention returns `ErrLocked`; callers choose whether and when to retry. Generated files are published through a temporary file and atomic rename. Factory failures leave no destination. Publication does not promise power-loss durability.

The factory runs under the lock and must not reenter for the same path. Existing files are returned unchanged; invalidation is caller-owned.

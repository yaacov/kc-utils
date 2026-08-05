---
name: architecture
description: >-
  Architectural design principles and guest conversion conventions for kc-utils:
  stage isolation, code blocks, plugin system, backend transparency, symlink
  paths, firstboot append safety, hypervisor cleanup helpers, RunInGuest
  patterns, and offline-vs-firstboot rules. Use before refactoring,
  splitting/merging packages, changing guest/backend boundaries, or modifying
  guest filesystem changes in convert-linux or convert-windows.
---

# kc-utils Architecture

Before changing code structure, packages, plugins, guest disk access, or
guest conversion filesystem changes:

1. Read [community/architecture.md](../../../community/architecture.md) with the Read tool for structural rules.
2. Read the "Guest conversion conventions" section in [community/CONTRIBUTING.md](../../../community/CONTRIBUTING.md) for guest filesystem change patterns (symlinks, firstboot, cleanup helpers, RunInGuest, offline removal).
3. Follow those rules for the change.
4. Prefer block/plugin isolation over shorter coupled code.

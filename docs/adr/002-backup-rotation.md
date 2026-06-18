# ADR-002: Backup Rotation Strategy

**Status**: Accepted  
**Date**: 2026-06-18  
**Deciders**: Lycaris (auditor)

## Context

The original code created a single backup at `config.sshady.bak`, overwriting
any previous backup. This meant only one restore point existed.

## Decision

Implement timestamped backups with automatic rotation:

- **Format**: `config.sshady.YYYYMMDD-HHMMSS.bak`
- **Rotation**: Keep last 5 backups, delete oldest
- **Non-blocking**: Rotation failure logs a warning but doesn't abort the write

Implementation:
- `createBackup()` writes timestamped backups
- `rotateBackups()` cleans up excess backups after each write
- `atomicWrite()` ensures the main config is never corrupted

## Alternatives Considered

- **git-based versioning**: Overkill for a config file; requires git repo in ~/.ssh
- **Single backup**: Original approach, insufficient for multiple changes
- **Infinite retention**: Risk of filling disk with backup files

## Consequences

- ~/.ssh/ directory accumulates up to 5 backup files
- Recovery is possible from any of the last 5 write operations
- Rotation is O(n) in number of files in ~/.ssh/ but n is small

# HelmShift v2 - Design Document

## Overview

HelmShift v2 is a complete redesign based on the principle: **migrations are fetched remotely, not stored locally**.

## Core Principles

1. **Remote Migration Registry** - Migrations stored in centralized registry, not local files or charts
2. **Zero External Dependencies** - Pure Go, no yq/jq/bash required
3. **Simple Declarative Format** - Easy-to-write operation-based transformations
4. **Automatic Path Finding** - Tool finds migration chain automatically (e.g., 0.1 → 0.2 → 0.3)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         User                                │
│                                                             │
│  helmshift -chart nginx-ingress -from 0.1.5 -to 0.3.0 \   │
│             -values values.yaml -registry https://...      │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                      CLI Layer                              │
│                  (cmd/helmshift)                            │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   Migrator (orchestration)                  │
│                  (pkg/migrator)                             │
│                                                             │
│  1. Fetch index from registry                              │
│  2. Find migration path (0.1.5 → 0.2.0 → 0.3.0)           │
│  3. Fetch each migration file                              │
│  4. Apply sequentially                                      │
└───────────┬────────────────────────────┬────────────────────┘
            │                            │
            ▼                            ▼
┌──────────────────────┐    ┌───────────────────────────────┐
│  Registry Client     │    │   Operations Engine           │
│  (pkg/registry)      │    │   (pkg/operations)            │
│                      │    │                               │
│  - HTTP fetcher      │    │  - Pure Go YAML manipulation  │
│  - Index parser      │    │  - Declarative operations     │
│  - Path finder       │    │  - Built-in transformations   │
└──────────────────────┘    └───────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────┐
│            Remote Migration Registry                        │
│         https://migrations-registry.io/                     │
│                                                             │
│  /nginx-ingress/                                           │
│    ├── index.yaml           # Migration index              │
│    ├── 0.1-to-0.2.yaml     # Migration file               │
│    └── 0.2-to-0.3.yaml     # Migration file               │
│                                                             │
│  /prometheus/                                              │
│    ├── index.yaml                                          │
│    └── ...                                                  │
└─────────────────────────────────────────────────────────────┘
```

## Migration Registry Structure

### Registry Layout

```
https://migrations-registry.io/
├── nginx-ingress/
│   ├── index.yaml
│   ├── 0.1-to-0.2.yaml
│   └── 0.2-to-0.3.yaml
├── prometheus/
│   ├── index.yaml
│   └── ...
└── [other-charts]/
```

### Index File Format

```yaml
# /nginx-ingress/index.yaml
chartName: nginx-ingress

migrations:
  - fromVersion: "0.1.x"      # Supports wildcards
    toVersion: "0.2.0"
    url: "/nginx-ingress/0.1-to-0.2.yaml"
    description: "Update image structure"

  - fromVersion: "0.2.x"
    toVersion: "0.3.0"
    url: "/nginx-ingress/0.2-to-0.3.yaml"
    description: "Move metrics under controller"
```

## Migration File Format

### Simple Declarative Operations

No code, just data:

```yaml
version: v1
chartName: nginx-ingress
fromVersion: "0.1.x"
toVersion: "0.2.0"
description: "Migration description"

operations:
  # Set a value
  - op: set
    path: controller.image.registry
    value: "registry.k8s.io"

  # Delete a value
  - op: delete
    path: controller.image.repository

  # Move a value (copy + delete source)
  - op: move
    from: metrics
    to: controller.metrics

  # Copy a value
  - op: copy
    from: controller.image.repository
    to: controller.image.image

  # Copy with transformation
  - op: copy
    from: controller.config.proxy-body-size
    to: controller.config.proxyBodySize
    transform:
      type: stripSuffix
      params:
        suffix: "-size"
```

### Supported Operations

| Operation | Description | Parameters |
|-----------|-------------|------------|
| `set` | Set value at path | `path`, `value` |
| `delete` | Delete value at path | `path` |
| `move` | Move value (copy + delete) | `from`, `to` |
| `copy` | Copy value | `from`, `to` |
| `rename` | Rename field (alias for move) | `from`, `to` |

### Supported Transformations

All transformations are pure Go (no external tools):

| Transformation | Description | Parameters |
|----------------|-------------|------------|
| `stripPrefix` | Remove prefix from string | `prefix` |
| `stripSuffix` | Remove suffix from string | `suffix` |
| `replace` | Replace substring | `old`, `new` |
| `toUpperCase` | Convert to uppercase | - |
| `toLowerCase` | Convert to lowercase | - |
| `toCamelCase` | Convert to camelCase | - |
| `toKebabCase` | Convert to kebab-case | - |
| `toSnakeCase` | Convert to snake_case | - |

## Workflow

### User Perspective

```bash
# Simple command - no local files needed
helmshift -chart nginx-ingress \
  -from 0.1.5 \
  -to 0.3.0 \
  -values values.yaml \
  -registry https://migrations-registry.io

# Output to file
helmshift -chart nginx-ingress \
  -from 0.1.5 \
  -to 0.3.0 \
  -values values.yaml \
  -output new-values.yaml
```

### Internal Flow

1. **Fetch Index**: GET `https://migrations-registry.io/nginx-ingress/index.yaml`
2. **Find Path**: Determine chain `0.1.5 → 0.2.0 → 0.3.0`
3. **Fetch Migrations**:
   - GET `/nginx-ingress/0.1-to-0.2.yaml`
   - GET `/nginx-ingress/0.2-to-0.3.yaml`
4. **Apply Sequentially**:
   - Parse original values.yaml
   - Apply 0.1 → 0.2 operations (pure Go)
   - Apply 0.2 → 0.3 operations (pure Go)
5. **Output**: Write migrated YAML

## Example Migration

### Input (0.1 values.yaml)

```yaml
controller:
  image:
    repository: k8s.gcr.io/ingress-nginx/controller
    tag: "v1.0.0"
  config:
    proxy-body-size: "10m"

metrics:
  enabled: true

defaultBackend:
  enabled: true
```

### After 0.1 → 0.2

```yaml
controller:
  image:
    registry: registry.k8s.io
    image: ingress-nginx/controller
    tag: "v1.0.0"
  config:
    proxyBodySize: "10m"

metrics:
  enabled: true

defaultBackend:
  enabled: true
```

### After 0.2 → 0.3 (Final)

```yaml
controller:
  image:
    registry: registry.k8s.io
    image: ingress-nginx/controller
    tag: "v1.0.0"
  config:
    proxyBodySize: "10m"
  metrics:
    enabled: true
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Migrate Helm values
  run: |
    helmshift \
      -chart nginx-ingress \
      -from 0.1.5 \
      -to 0.3.0 \
      -values values.yaml \
      -output migrated-values.yaml \
      -registry https://migrations-registry.io
```

### GitLab CI

```yaml
migrate:
  script:
    - helmshift -chart nginx-ingress -from 0.1.5 -to 0.3.0 -values values.yaml
```

## Advantages Over V1

| Aspect | V1 (Old) | V2 (New) |
|--------|----------|----------|
| **Migration Storage** | Local files | Remote registry |
| **Dependencies** | yq, jq, bash | None (pure Go) |
| **Format Complexity** | 3 formats (JSON Patch, yq, scripts) | 1 simple format |
| **CI/CD Setup** | Copy migration files | Just run tool |
| **Discoverability** | Manual file management | Auto-fetch from registry |
| **Versioning** | Manual tracking | Index-based |
| **Learning Curve** | High (3 formats, tools) | Low (simple operations) |

## Registry Hosting Options

### Option 1: GitHub Pages (Free)

```
https://username.github.io/helm-migrations/
├── nginx-ingress/
│   ├── index.yaml
│   └── ...
```

### Option 2: OCI Registry (Future)

```
oci://ghcr.io/username/helm-migrations/nginx-ingress:latest
```

### Option 3: S3/GCS (Cloud)

```
https://migrations.s3.amazonaws.com/
```

### Option 4: Chart Repo (Existing Infrastructure)

```
https://charts.example.com/migrations/
```

## Who Maintains Migrations?

### Recommended Approach

1. **Chart Maintainers**: Publish migrations alongside chart releases
2. **Registry**: Community-run or org-specific
3. **Versioning**: Migration files versioned with chart releases
4. **Discovery**: Tool uses `-registry` flag (or default)

### Example Workflow

```bash
# Chart maintainer releases v0.2.0
1. Create migration: 0.1-to-0.2.yaml
2. Update index.yaml
3. Publish to registry

# User upgrades chart
helmshift -chart mychart -from 0.1.5 -to 0.2.0 -values values.yaml
# Tool automatically fetches latest migration from registry
```

## Future Enhancements

1. **OCI Support**: Store migrations in OCI registries
2. **Signature Verification**: Sign migrations for security
3. **Helm Plugin**: `helm migrate` command
4. **Migration Testing**: Validate migrations before applying
5. **Dry-Run Diff**: Show what will change before applying
6. **Rollback**: Reverse migrations (if possible)
7. **Local Cache**: Cache downloaded migrations

## Implementation Status

- ✅ Remote registry client
- ✅ Index fetching and parsing
- ✅ Migration path finding
- ✅ Pure Go operation engine
- ✅ All transformation types
- ✅ CLI interface
- ✅ Example migrations
- ⏳ Helm plugin wrapper
- ⏳ Documentation
- ⏳ Testing

## Summary

HelmShift v2 solves the original problems:

1. ❌ **Local files required** → ✅ **Fetch from remote registry**
2. ❌ **External tool dependencies** → ✅ **Pure Go, zero dependencies**
3. ❌ **Complex formats** → ✅ **Simple declarative operations**
4. ❌ **Manual migration chains** → ✅ **Automatic path finding**

The tool is now truly CI/CD-ready with minimal setup and maximum flexibility.

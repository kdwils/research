# HelmShift

> Automate Helm values migration when chart schemas change

HelmShift is a flexible CLI tool that automates the migration of Helm `values.yaml` files when charts introduce breaking changes between versions. It supports multiple patch formats to accommodate different use cases and team preferences.

## Features

- ✅ **Multiple Patch Formats**: JSON Patch (RFC 6902), yq expressions, and executable scripts
- ✅ **User-Defined Migrations**: Patches are authored and stored separately from the tool
- ✅ **CI/CD Native**: Designed for automation pipelines with dry-run mode
- ✅ **Validation**: Pre and post-migration validation with custom rules
- ✅ **Standalone**: No dependencies on Helm or Kubernetes
- ✅ **Platform Agnostic**: Works with any CI/CD platform

## Quick Start

### Installation

#### Download Binary
```bash
VERSION="0.1.0"
wget https://github.com/user/helmshift/releases/download/v${VERSION}/helmshift-linux-amd64
chmod +x helmshift-linux-amd64
sudo mv helmshift-linux-amd64 /usr/local/bin/helmshift
```

#### Build from Source
```bash
git clone https://github.com/kdwils/helmshift.git
cd helmshift
go build -o helmshift ./cmd/helmshift
```

### Basic Usage

1. **Create a migration configuration**:

```yaml
# migration.yaml
fromVersion: "3.x"
toVersion: "4.0"
format: jsonpatch
description: "Migrate nginx-ingress from v3 to v4"
patchFile: v3-to-v4.jsonpatch.yaml

validation:
  requiredFields:
    - controller
  forbiddenFields:
    - defaultBackend
```

2. **Create a patch file**:

```yaml
# v3-to-v4.jsonpatch.yaml
- op: add
  path: /controller/image/registry
  value: registry.k8s.io

- op: remove
  path: /controller/image/repository

- op: move
  from: /metrics
  path: /controller/metrics
```

3. **Run the migration**:

```bash
# Dry-run (validate only)
helmshift -config migration.yaml -values values.yaml -dry-run

# Migrate and output to stdout
helmshift -config migration.yaml -values values.yaml

# Migrate and save to file
helmshift -config migration.yaml -values values.yaml -output new-values.yaml
```

## Patch Formats

HelmShift supports three patch formats:

### 1. JSON Patch (RFC 6902)

**Best for**: Standard, well-defined transformations

```yaml
# Format: jsonpatch
- op: add
  path: /new/field
  value: "value"

- op: remove
  path: /old/field

- op: replace
  path: /field
  value: "new value"

- op: move
  from: /old/location
  path: /new/location
```

### 2. yq Expressions

**Best for**: YAML-native transformations, structural changes

```yaml
# Format: yq
expressions:
  - '.new.field = "value"'
  - 'del(.old.field)'
  - '.new.location = .old.location'
  - 'del(.old.location)'
```

### 3. Executable Scripts

**Best for**: Complex logic, custom transformations

```bash
#!/bin/bash
# Format: script

# Read from stdin, write to stdout
yq eval '.new.field = "value"' |
yq eval 'del(.old.field)' |
yq eval '.new.location = .old.location' |
yq eval 'del(.old.location)'
```

See [docs/patch-formats.md](docs/patch-formats.md) for detailed specifications.

## Examples

### Example: Migrating nginx-ingress v3 → v4

The repository includes a complete example migrating the nginx-ingress chart from v3 to v4:

```bash
# Using JSON Patch
./helmshift \
  -config examples/patches/nginx-ingress/jsonpatch-migration.yaml \
  -values examples/values/nginx-ingress/v3-values.yaml \
  -output v4-values.yaml

# Using script-based migration
./helmshift \
  -config examples/patches/nginx-ingress/script-migration.yaml \
  -values examples/values/nginx-ingress/v3-values.yaml \
  -output v4-values.yaml

# Using yq expressions
./helmshift \
  -config examples/patches/nginx-ingress/yq-migration.yaml \
  -values examples/values/nginx-ingress/v3-values.yaml \
  -output v4-values.yaml
```

### Example Migration Changes

The example demonstrates common migration scenarios:

- **Image repository structure change**: Split `repository` into `registry` and `image`
- **Field naming convention change**: Rename from `kebab-case` to `camelCase`
- **Configuration restructuring**: Move `metrics` from root to `controller.metrics`
- **Deprecated feature removal**: Remove `defaultBackend` configuration

## CI/CD Integration

HelmShift works seamlessly with all CI/CD platforms:

### GitHub Actions

```yaml
- name: Migrate values
  run: |
    helmshift \
      -config migrations/v3-to-v4.yaml \
      -values values.yaml \
      -output new-values.yaml
```

### GitLab CI

```yaml
migrate:
  script:
    - helmshift -config migrations/v3-to-v4.yaml -values values.yaml -output new-values.yaml
```

### Jenkins

```groovy
sh 'helmshift -config migrations/v3-to-v4.yaml -values values.yaml -output new-values.yaml'
```

See [docs/ci-integration.md](docs/ci-integration.md) for comprehensive integration patterns.

## CLI Reference

```
helmshift [options]

Options:
  -config string      Path to migration configuration file (required)
  -values string      Path to values.yaml file to migrate (required)
  -output string      Path to write migrated values (default: stdout)
  -dry-run            Perform migration but don't write output
  -timeout duration   Timeout for migration execution (default 30s)
  -version            Show version information
```

### Exit Codes

- `0`: Success
- `1`: Error (migration failed, validation failed, etc.)

## Project Structure

```
helmshift/
├── cmd/
│   └── helmshift/          # CLI entry point
├── pkg/
│   ├── config/             # Migration configuration
│   ├── migration/          # Migration engine
│   │   └── formats/        # Patch format implementations
│   └── parser/             # YAML parsing utilities
├── examples/
│   ├── patches/            # Example migration patches
│   │   └── nginx-ingress/
│   └── values/             # Example values files
│       └── nginx-ingress/
├── docs/
│   ├── research.md         # Research & design document
│   ├── ci-integration.md   # CI/CD integration guide
│   └── patch-formats.md    # Patch format specifications
└── README.md
```

## Documentation

- [Research & Design](docs/research.md) - Comprehensive analysis of existing tools and solution design
- [CI/CD Integration](docs/ci-integration.md) - Integration patterns for various CI/CD platforms
- [Patch Formats](docs/patch-formats.md) - Detailed patch format specifications

## Comparison with Alternatives

| Feature | HelmShift | helm-migrate-values | Manual Scripts |
|---------|-----------|---------------------|----------------|
| Multiple formats | ✅ | ❌ | ✅ |
| Standalone tool | ✅ | ❌ (plugin) | ✅ |
| Validation | ✅ | ❌ | ❌ |
| Dry-run mode | ✅ | ⚠️ | ❌ |
| Easy to learn | ✅ | ❌ | ⚠️ |
| No cluster needed | ✅ | ❌ | ✅ |

## Limitations (PoC)

This is a proof-of-concept implementation with some limitations:

1. **yq Format**: Simplified implementation (full yq library integration needed for production)
2. **Validation**: Basic field-level checks (could be extended with JSON Schema)
3. **Testing**: Manual testing only (unit/integration tests needed)
4. **Performance**: Not optimized for very large values files

See [docs/research.md](docs/research.md#proof-of-concept-limitations) for details.

## Future Enhancements

- [ ] Full yq library integration
- [ ] JSON Schema validation support
- [ ] Multi-step migration chains (v1→v2→v3 in one command)
- [ ] Diff visualization
- [ ] Interactive migration wizard
- [ ] Helm plugin wrapper
- [ ] Rollback capability

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

Apache 2.0

## Acknowledgments

- Inspired by [helm-migrate-values](https://github.com/OctopusDeployLabs/helm-migrate-values)
- Uses [evanphx/json-patch](https://github.com/evanphx/json-patch) for JSON Patch support
- Built with Go and love for automation

## Support

- 📖 [Documentation](docs/)
- 🐛 [Issue Tracker](https://github.com/kdwils/helmshift/issues)
- 💬 [Discussions](https://github.com/kdwils/helmshift/discussions)

---

**Made with ❤️ for the Helm community**

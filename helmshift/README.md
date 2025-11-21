# HelmShift

> Automate Helm values migration with zero dependencies

HelmShift is a CLI tool that automates the migration of Helm `values.yaml` files when charts introduce breaking changes between versions. Migrations are fetched from a remote registry and applied using pure Go - no external tools required.

## ✨ Key Features

- 🌐 **Remote Migration Registry** - Migrations fetched from central registry, no local files needed
- 🚀 **Zero Dependencies** - Pure Go implementation, no yq/jq/bash required
- 📝 **Simple Declarative Format** - Easy-to-write operation-based transformations
- 🔄 **Automatic Path Finding** - Migrates through multiple versions automatically (0.1 → 0.2 → 0.3)
- 🔧 **CI/CD Ready** - Designed for automation pipelines
- ✅ **Type-Safe** - Compile-time guarantees, no runtime script errors

## 🎯 Design Philosophy

**Problem**: Previous solutions required either:
- Local migration files (needs file management in CI/CD)
- External tool dependencies (yq, jq, bash scripts)
- Migrations embedded in charts (requires pulling intermediate chart versions)

**HelmShift Solution**:
1. Migrations stored in **remote registry** (fetch on-demand)
2. **Pure Go execution** (no external tools)
3. **Simple declarative operations** (no code/templates)

See [DESIGN-V2.md](./DESIGN-V2.md) for complete architecture.

## 📥 Installation

### Download Binary

```bash
VERSION="0.2.0"
wget https://github.com/user/helmshift/releases/download/v${VERSION}/helmshift-linux-amd64
chmod +x helmshift-linux-amd64
sudo mv helmshift-linux-amd64 /usr/local/bin/helmshift
```

### Build from Source

```bash
git clone https://github.com/kdwils/helmshift.git
cd helmshift
go build -o helmshift ./cmd/helmshift
```

## 🚀 Quick Start

```bash
# Migrate nginx-ingress from v0.1.5 to v0.3.0
helmshift -chart nginx-ingress \
  -from 0.1.5 \
  -to 0.3.0 \
  -values values.yaml \
  -registry https://migrations-registry.io

# Save to file
helmshift -chart nginx-ingress \
  -from 0.1.5 \
  -to 0.3.0 \
  -values values.yaml \
  -output new-values.yaml
```

That's it! No local migration files needed.

## 📖 How It Works

### 1. User Runs Command

```bash
helmshift -chart nginx-ingress -from 0.1.5 -to 0.3.0 -values values.yaml
```

### 2. Tool Fetches Migration Index

```
GET https://migrations-registry.io/nginx-ingress/index.yaml
```

Returns:
```yaml
chartName: nginx-ingress
migrations:
  - fromVersion: "0.1.x"
    toVersion: "0.2.0"
    url: "/nginx-ingress/0.1-to-0.2.yaml"
  - fromVersion: "0.2.x"
    toVersion: "0.3.0"
    url: "/nginx-ingress/0.2-to-0.3.yaml"
```

### 3. Tool Finds Migration Path

Determines: `0.1.5 → 0.2.0 → 0.3.0`

### 4. Tool Fetches & Applies Migrations

```
GET /nginx-ingress/0.1-to-0.2.yaml  →  Apply operations
GET /nginx-ingress/0.2-to-0.3.yaml  →  Apply operations
```

### 5. Output Migrated Values

All transformations done in pure Go - no external tools!

## 📝 Migration Format

Simple, declarative operations (no code required):

```yaml
version: v1
chartName: nginx-ingress
fromVersion: "0.1.x"
toVersion: "0.2.0"
description: "Migrate nginx-ingress from v0.1 to v0.2"

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

  # Copy with transformation
  - op: copy
    from: controller.config.proxy-body-size
    to: controller.config.proxyBodySize
    transform:
      type: toCamelCase
```

### Supported Operations

| Operation | Description |
|-----------|-------------|
| `set` | Set value at path |
| `delete` | Delete value at path |
| `move` | Move value (copy + delete source) |
| `copy` | Copy value |

### Built-in Transformations

All pure Go (no external tools):

- `stripPrefix` / `stripSuffix`
- `replace`
- `toUpperCase` / `toLowerCase`
- `toCamelCase` / `toKebabCase` / `toSnakeCase`

## 🏗️ Migration Registry

### Structure

```
https://migrations-registry.io/
├── nginx-ingress/
│   ├── index.yaml          # Migration index
│   ├── 0.1-to-0.2.yaml    # Migration operations
│   └── 0.2-to-0.3.yaml
├── prometheus/
│   ├── index.yaml
│   └── ...
└── [other-charts]/
```

### Hosting Options

- **GitHub Pages** (free)
- **S3/GCS** (cloud storage)
- **Existing Chart Repo** (reuse infrastructure)
- **OCI Registry** (future support)

## 🔧 CI/CD Integration

### GitHub Actions

```yaml
- name: Migrate Helm values
  run: |
    helmshift \
      -chart nginx-ingress \
      -from ${{ env.CURRENT_VERSION }} \
      -to ${{ env.TARGET_VERSION }} \
      -values values.yaml \
      -output migrated-values.yaml
```

### GitLab CI

```yaml
migrate:
  script:
    - helmshift -chart nginx-ingress -from 0.1.5 -to 0.3.0 -values values.yaml -output new-values.yaml
```

### Jenkins

```groovy
sh 'helmshift -chart nginx-ingress -from 0.1.5 -to 0.3.0 -values values.yaml'
```

No special setup needed - just run the binary!

## 📊 Examples

See [examples/registry/](./examples/registry/) for:
- Example migration registry structure
- nginx-ingress migrations (0.1 → 0.2 → 0.3)
- Sample values files

## 🆚 Comparison

| Feature | HelmShift v2 | Manual Migration | helm-migrate-values |
|---------|--------------|------------------|---------------------|
| Local files required | ❌ | ✅ | ✅ |
| External tools (yq/jq) | ❌ | ✅ | ❌ |
| Kubernetes cluster | ❌ | ❌ | ✅ |
| Multi-version jump | ✅ | ❌ | ✅ |
| Pure Go | ✅ | ❌ | ❌ |
| Learning curve | Low | High | Medium |

## 📚 Documentation

- [DESIGN-V2.md](./DESIGN-V2.md) - Complete architecture and design decisions
- [examples/](./examples/) - Sample migrations and registry structure

## 🤝 Contributing

1. Fork the repository
2. Create feature branch
3. Add tests
4. Submit pull request

## 📄 License

Apache 2.0

## 🙏 Acknowledgments

- Inspired by the need for better Helm migration tooling
- Built with pure Go for maximum portability

---

**Made with ❤️ for the Helm community**

*Automate your migrations, simplify your life.*

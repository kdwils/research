# Patch Format Specifications

This document provides detailed specifications for all patch formats supported by HelmShift.

## Table of Contents

1. [Overview](#overview)
2. [JSON Patch (RFC 6902)](#json-patch-rfc-6902)
3. [yq Expressions](#yq-expressions)
4. [Executable Scripts](#executable-scripts)
5. [Choosing a Format](#choosing-a-format)
6. [Best Practices](#best-practices)

---

## Overview

HelmShift supports three patch formats, each with different strengths:

| Format | Use Case | Complexity | Portability |
|--------|----------|------------|-------------|
| JSON Patch | Standard operations | Low | High |
| yq | YAML transformations | Medium | Medium |
| Script | Complex logic | High | Low |

All formats:
- Read YAML values as input
- Output YAML values
- Support validation
- Work in CI/CD pipelines

---

## JSON Patch (RFC 6902)

### Overview

JSON Patch is an [IETF standard](https://datatracker.ietf.org/doc/html/rfc6902) for describing changes to JSON documents. HelmShift accepts patches in YAML format and applies them to YAML values files.

### Migration Configuration

```yaml
fromVersion: "1.0"
toVersion: "2.0"
format: jsonpatch
description: "Description of migration"
patchFile: migration.jsonpatch.yaml
```

### Operations

#### 1. Add

Add a new field or value.

```yaml
- op: add
  path: /new/field
  value: "new value"
```

**Examples**:

```yaml
# Add simple value
- op: add
  path: /controller/replicas
  value: 3

# Add object
- op: add
  path: /controller/resources
  value:
    limits:
      cpu: 500m
      memory: 512Mi

# Add to array (end)
- op: add
  path: /environments/-
  value: "production"

# Add to array (specific index)
- op: add
  path: /environments/0
  value: "development"
```

#### 2. Remove

Delete a field.

```yaml
- op: remove
  path: /deprecated/field
```

**Examples**:

```yaml
# Remove simple field
- op: remove
  path: /oldConfig

# Remove nested field
- op: remove
  path: /controller/deprecated/setting

# Remove array element
- op: remove
  path: /environments/2
```

#### 3. Replace

Update an existing value.

```yaml
- op: replace
  path: /existing/field
  value: "updated value"
```

**Examples**:

```yaml
# Replace string
- op: replace
  path: /image/tag
  value: "v2.0.0"

# Replace number
- op: replace
  path: /replicas
  value: 5

# Replace object
- op: replace
  path: /resources
  value:
    requests:
      cpu: 100m
```

#### 4. Move

Relocate a field.

```yaml
- op: move
  from: /old/location
  path: /new/location
```

**Examples**:

```yaml
# Move field to new location
- op: move
  from: /metrics
  path: /controller/metrics

# Rename field (move within same parent)
- op: move
  from: /config/old-name
  path: /config/new-name
```

#### 5. Copy

Duplicate a field.

```yaml
- op: copy
  from: /source/field
  path: /destination/field
```

**Examples**:

```yaml
# Copy configuration to another section
- op: copy
  from: /defaults/config
  path: /controller/config

# Duplicate array
- op: copy
  from: /commonLabels
  path: /controller/labels
```

#### 6. Test

Assert a value (validation).

```yaml
- op: test
  path: /field
  value: "expected value"
```

**Examples**:

```yaml
# Verify value before migration
- op: test
  path: /version
  value: "3.0"

# Test then modify
- op: test
  path: /enabled
  value: true
- op: replace
  path: /enabled
  value: false
```

### Path Syntax

Paths use JSON Pointer syntax ([RFC 6901](https://datatracker.ietf.org/doc/html/rfc6901)):

```yaml
# Root level
path: /field

# Nested
path: /parent/child/grandchild

# Array index
path: /items/0

# Append to array
path: /items/-

# Escape special characters
path: /field~0with~1slashes  # field~with/slashes
```

### Complete Example

```yaml
# migration.jsonpatch.yaml
# Migrate nginx-ingress v3 → v4

# 1. Update image structure
- op: add
  path: /controller/image/registry
  value: "registry.k8s.io"

- op: add
  path: /controller/image/image
  value: "ingress-nginx/controller"

- op: remove
  path: /controller/image/repository

# 2. Rename config keys
- op: move
  from: /controller/config/use-forwarded-headers
  path: /controller/config/useForwardedHeaders

- op: move
  from: /controller/config/proxy-body-size
  path: /controller/config/proxyBodySize

# 3. Restructure metrics
- op: move
  from: /metrics
  path: /controller/metrics

# 4. Remove deprecated features
- op: remove
  path: /defaultBackend

# 5. Update version
- op: replace
  path: /version
  value: "4.0"
```

### Validation

```yaml
# With validation rules
fromVersion: "3.x"
toVersion: "4.0"
format: jsonpatch
patchFile: migration.jsonpatch.yaml

validation:
  requiredFields:
    - controller
    - controller/image/registry
    - controller/image/image
  forbiddenFields:
    - defaultBackend
    - controller/image/repository
```

---

## yq Expressions

### Overview

yq is a powerful YAML processor. HelmShift uses yq-style expressions to transform values files.

### Migration Configuration

```yaml
fromVersion: "1.0"
toVersion: "2.0"
format: yq
description: "Description of migration"
patchFile: migration.yq.yaml
```

### Expression Format

```yaml
# migration.yq.yaml
expressions:
  - 'expression 1'
  - 'expression 2'
  - 'expression 3'
```

Expressions are applied sequentially.

### Common Operations

#### Set a field

```yaml
expressions:
  - '.controller.replicas = 3'
  - '.image.tag = "v2.0.0"'
  - '.enabled = true'
```

#### Delete a field

```yaml
expressions:
  - 'del(.deprecated)'
  - 'del(.controller.oldConfig)'
```

#### Copy/Rename field

```yaml
expressions:
  # Copy
  - '.new = .old'

  # Rename (copy + delete)
  - '.newName = .oldName'
  - 'del(.oldName)'
```

#### Conditional operations

```yaml
expressions:
  # Set if exists
  - '.controller.enabled //= true'

  # Conditional transformation
  - 'if .type == "LoadBalancer" then .annotations.external = true else . end'
```

#### Array operations

```yaml
expressions:
  # Add to array
  - '.items += ["new-item"]'

  # Filter array
  - '.items = .items | select(. != "unwanted")'

  # Map over array
  - '.items = .items | map(. + "-suffix")'
```

#### Nested updates

```yaml
expressions:
  # Update nested value
  - '.controller.resources.limits.cpu = "500m"'

  # Create nested structure
  - '.controller.new.nested.field = "value"'
```

### Complete Example

```yaml
# migration.yq.yaml
# Migrate nginx-ingress v3 → v4

expressions:
  # 1. Update image structure
  - '.controller.image.registry = "registry.k8s.io"'
  - '.controller.image.image = "ingress-nginx/controller"'
  - 'del(.controller.image.repository)'

  # 2. Rename config keys (kebab to camel)
  - '.controller.config.useForwardedHeaders = .controller.config."use-forwarded-headers"'
  - 'del(.controller.config."use-forwarded-headers")'

  - '.controller.config.proxyBodySize = .controller.config."proxy-body-size"'
  - 'del(.controller.config."proxy-body-size")'

  # 3. Move metrics
  - '.controller.metrics = .metrics'
  - 'del(.metrics)'

  # 4. Remove deprecated
  - 'del(.defaultBackend)'

  # 5. Update version
  - '.version = "4.0"'
```

### Advanced Examples

#### Transforming all keys

```yaml
expressions:
  # Convert all keys to camelCase (simplified)
  - |
    walk(
      if type == "object" then
        with_entries(.key |= gsub("-"; ""))
      else . end
    )
```

#### Merging configurations

```yaml
expressions:
  # Merge old config into new structure
  - '.controller.config = (.controller.config + .legacy.config)'
  - 'del(.legacy)'
```

#### Complex restructuring

```yaml
expressions:
  # Transform structure
  - |
    .controller.services = [
      {
        "name": "http",
        "port": .controller.service.ports.http
      },
      {
        "name": "https",
        "port": .controller.service.ports.https
      }
    ]
  - 'del(.controller.service.ports)'
```

### Notes

**PoC Limitation**: The current implementation has a simplified yq expression evaluator. For production use, full integration with `github.com/mikefarah/yq/v4` is recommended.

---

## Executable Scripts

### Overview

Scripts provide maximum flexibility by allowing any transformation logic. The script reads YAML from stdin and writes YAML to stdout.

### Migration Configuration

```yaml
fromVersion: "1.0"
toVersion: "2.0"
format: script
description: "Description of migration"
patchFile: migration.sh
```

### Script Requirements

1. **Input**: Read YAML from stdin
2. **Output**: Write YAML to stdout
3. **Exit code**: 0 for success, non-zero for failure
4. **Errors**: Write to stderr

### Supported Interpreters

Scripts are automatically interpreted based on file extension:

- `.sh`, `.bash`: Bash
- `.py`: Python 3
- `.rb`: Ruby
- `.js`: Node.js
- `.pl`: Perl
- Executable files with shebang

### Bash Script Example

```bash
#!/bin/bash
# migration.sh

set -euo pipefail

# For simple transformations, use yq
yq eval '
  .controller.image.registry = "registry.k8s.io" |
  .controller.image.image = "ingress-nginx/controller" |
  del(.controller.image.repository) |
  .controller.metrics = .metrics |
  del(.metrics) |
  del(.defaultBackend)
'
```

### Python Script Example

```python
#!/usr/bin/env python3
# migration.py

import sys
import yaml

# Read input
data = yaml.safe_load(sys.stdin)

# Transform
controller = data.get('controller', {})
image = controller.get('image', {})

# Split repository
if 'repository' in image:
    image['registry'] = 'registry.k8s.io'
    image['image'] = 'ingress-nginx/controller'
    del image['repository']

# Move metrics
if 'metrics' in data:
    controller['metrics'] = data['metrics']
    del data['metrics']

# Remove deprecated
if 'defaultBackend' in data:
    del data['defaultBackend']

# Output
yaml.dump(data, sys.stdout, default_flow_style=False)
```

### Advanced Bash Script

```bash
#!/bin/bash
# advanced-migration.sh

set -euo pipefail

# Read input to temp file
INPUT=$(mktemp)
cat > "$INPUT"

# Function to check if field exists
field_exists() {
    yq eval "has(\"$1\")" "$INPUT" | grep -q "true"
}

# Conditional transformations
if field_exists "legacy"; then
    yq eval '.controller.config = (.controller.config + .legacy.config)' "$INPUT" |
    yq eval 'del(.legacy)' > "$INPUT.tmp"
    mv "$INPUT.tmp" "$INPUT"
fi

# Standard transformations
yq eval '
  .controller.image.registry = "registry.k8s.io" |
  .controller.image.image = "ingress-nginx/controller" |
  del(.controller.image.repository)
' "$INPUT"

# Cleanup
rm -f "$INPUT"
```

### Script with External Tools

```bash
#!/bin/bash
# migration-with-validation.sh

set -euo pipefail

# Read input
INPUT=$(mktemp)
cat > "$INPUT"

# Validate input
if ! yq eval '.version' "$INPUT" | grep -q "3"; then
    echo "Error: Expected version 3.x" >&2
    exit 1
fi

# Transform
OUTPUT=$(yq eval '
  .controller.image.registry = "registry.k8s.io" |
  .controller.image.image = "ingress-nginx/controller" |
  del(.controller.image.repository) |
  .version = "4.0"
' "$INPUT")

# Validate output
if ! echo "$OUTPUT" | yq eval '.controller.image.registry' | grep -q "registry.k8s.io"; then
    echo "Error: Migration validation failed" >&2
    exit 1
fi

# Output
echo "$OUTPUT"

# Cleanup
rm -f "$INPUT"
```

### Ruby Script Example

```ruby
#!/usr/bin/env ruby
# migration.rb

require 'yaml'

# Read input
data = YAML.load($stdin)

# Transform
controller = data['controller'] ||= {}
image = controller['image'] ||= {}

# Update image
if image['repository']
  image['registry'] = 'registry.k8s.io'
  image['image'] = 'ingress-nginx/controller'
  image.delete('repository')
end

# Move metrics
if data['metrics']
  controller['metrics'] = data['metrics']
  data.delete('metrics')
end

# Remove deprecated
data.delete('defaultBackend')

# Output
puts YAML.dump(data)
```

### Error Handling

```bash
#!/bin/bash
# migration-with-errors.sh

set -euo pipefail

INPUT=$(mktemp)
cat > "$INPUT"

# Validate input exists
if [ ! -s "$INPUT" ]; then
    echo "Error: Empty input" >&2
    exit 1
fi

# Try transformation with error handling
if ! OUTPUT=$(yq eval '.controller.image.registry = "registry.k8s.io"' "$INPUT" 2>&1); then
    echo "Error: yq transformation failed: $OUTPUT" >&2
    rm -f "$INPUT"
    exit 1
fi

echo "$OUTPUT"
rm -f "$INPUT"
```

### Testing Scripts Standalone

```bash
# Test script independently
cat values.yaml | ./migration.sh > output.yaml

# Validate output
yq eval '.' output.yaml  # Check valid YAML
diff -u values.yaml output.yaml  # See changes
```

---

## Choosing a Format

### Decision Matrix

```
┌─────────────────────────────────────────────────────┐
│ Start: What type of migration?                     │
└───────────────┬─────────────────────────────────────┘
                │
        ┌───────┴───────┐
        │ Simple field  │
        │ add/remove/   │
        │ move?         │
        └───────┬───────┘
                │
        ┌───────┴───────┐
        ▼ YES           ▼ NO
┌───────────────┐   ┌───────────────┐
│  JSON Patch   │   │ Complex/      │
│               │   │ conditional?  │
└───────────────┘   └───────┬───────┘
                            │
                    ┌───────┴───────┐
                    ▼ YES           ▼ NO
            ┌───────────────┐   ┌───────────────┐
            │    Script     │   │ yq            │
            │               │   │ Expressions   │
            └───────────────┘   └───────────────┘
```

### Recommendations

| Scenario | Format | Reason |
|----------|--------|--------|
| Field renames | JSON Patch or yq | Standard operations |
| Value updates | JSON Patch | Clear semantics |
| Structural changes | yq | YAML-native |
| Conditional logic | Script | Flexibility |
| Array transformations | yq | Array operations |
| External API calls | Script | Can use curl/wget |
| Complex validation | Script | Custom logic |
| Team unfamiliar with tools | JSON Patch | Well-documented |
| YAML-first team | yq | Familiar syntax |
| Maximum compatibility | JSON Patch | Standard (RFC 6902) |

---

## Best Practices

### 1. Keep Patches Simple

```yaml
# Good: One logical change per operation
- op: move
  from: /metrics
  path: /controller/metrics

# Avoid: Combining unrelated changes
```

### 2. Add Comments

```yaml
# migration.jsonpatch.yaml

# Step 1: Update image structure for new registry
- op: add
  path: /controller/image/registry
  value: "registry.k8s.io"

# Step 2: Remove deprecated backend
- op: remove
  path: /defaultBackend
```

### 3. Validate Early

```yaml
# Test expected value first
- op: test
  path: /version
  value: "3.0"

# Then make changes
- op: replace
  path: /version
  value: "4.0"
```

### 4. Sequence Matters

```yaml
# Correct order: copy then delete
- op: copy
  from: /old
  path: /new
- op: remove
  path: /old

# Wrong order: delete then copy (fails!)
# - op: remove
#   path: /old
# - op: copy
#   from: /old  # <-- old doesn't exist anymore!
#   path: /new
```

### 5. Use Validation Rules

```yaml
# migration.yaml
validation:
  requiredFields:
    - controller
    - controller/image/registry
  forbiddenFields:
    - defaultBackend
    - controller/image/repository
```

### 6. Test Thoroughly

```bash
# Test on sample values
helmshift -config migration.yaml -values test-values.yaml -dry-run

# Validate output
helmshift -config migration.yaml -values test-values.yaml | yq eval '.'

# Check specific fields
helmshift -config migration.yaml -values test-values.yaml | \
  yq eval '.controller.image.registry'
```

### 7. Document Breaking Changes

```yaml
# migration.yaml
fromVersion: "3.x"
toVersion: "4.0"
format: jsonpatch
description: |
  Breaking changes:
  - Image repository split into registry + image
  - Config keys renamed to camelCase
  - Metrics moved under controller
  - defaultBackend removed (use ingress-nginx default)

  Migration steps:
  1. Update image structure
  2. Rename configuration keys
  3. Restructure metrics
  4. Remove deprecated features
patchFile: v3-to-v4.jsonpatch.yaml
```

### 8. Version Control Patches

```bash
# Store patches with chart version
migrations/
├── v1-to-v2/
│   ├── migration.yaml
│   └── patch.jsonpatch.yaml
├── v2-to-v3/
│   ├── migration.yaml
│   └── patch.yq.yaml
└── v3-to-v4/
    ├── migration.yaml
    └── patch.sh
```

---

## Appendix

### JSON Patch Resources

- [RFC 6902 Specification](https://datatracker.ietf.org/doc/html/rfc6902)
- [JSON Patch Website](https://jsonpatch.com/)
- [evanphx/json-patch Library](https://github.com/evanphx/json-patch)

### yq Resources

- [yq Documentation](https://mikefarah.gitbook.io/yq)
- [yq GitHub](https://github.com/mikefarah/yq)
- [yq Expression Guide](https://mikefarah.gitbook.io/yq/operators)

### Script Resources

- [Bash Best Practices](https://google.github.io/styleguide/shellguide.html)
- [YAML Python Library](https://pyyaml.org/)
- [Ruby YAML](https://ruby-doc.org/stdlib-3.0.0/libdoc/yaml/rdoc/YAML.html)

#!/bin/bash
# Script-based migration for nginx-ingress v3 to v4
# This script reads values from stdin and outputs migrated values to stdout

set -euo pipefail

# Read input from stdin into a temporary file
INPUT=$(mktemp)
cat > "$INPUT"

# For PoC, we'll use yq (if available) or simple sed transformations
if command -v yq &> /dev/null; then
    # Use yq for sophisticated YAML transformations

    # 1. Split image.repository into registry and image
    yq eval '.controller.image.registry = "registry.k8s.io"' "$INPUT" |
    yq eval '.controller.image.image = "ingress-nginx/controller"' |
    yq eval 'del(.controller.image.repository)' |

    # 2. Rename config keys (kebab-case to camelCase)
    yq eval '.controller.config.useForwardedHeaders = .controller.config."use-forwarded-headers"' |
    yq eval 'del(.controller.config."use-forwarded-headers")' |
    yq eval '.controller.config.computeFullForwardedFor = .controller.config."compute-full-forwarded-for"' |
    yq eval 'del(.controller.config."compute-full-forwarded-for")' |
    yq eval '.controller.config.proxyBodySize = .controller.config."proxy-body-size"' |
    yq eval 'del(.controller.config."proxy-body-size")' |

    # 3. Move metrics to controller.metrics
    yq eval '.controller.metrics = .metrics' |
    yq eval 'del(.metrics)' |

    # 4. Remove deprecated defaultBackend
    yq eval 'del(.defaultBackend)'
else
    # Fallback: Simple transformations using sed/awk
    # This is a simplified version for demonstration
    cat "$INPUT" | sed \
        -e 's/repository: k8s.gcr.io\/ingress-nginx\/controller/registry: registry.k8s.io\n    image: ingress-nginx\/controller/' \
        -e 's/use-forwarded-headers/useForwardedHeaders/g' \
        -e 's/compute-full-forwarded-for/computeFullForwardedFor/g' \
        -e 's/proxy-body-size/proxyBodySize/g'
fi

# Clean up
rm -f "$INPUT"

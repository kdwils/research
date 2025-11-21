# CI/CD Integration Patterns for HelmShift

This document provides comprehensive patterns and examples for integrating HelmShift into various CI/CD platforms.

## Table of Contents

1. [General Integration Principles](#general-integration-principles)
2. [GitHub Actions](#github-actions)
3. [GitLab CI/CD](#gitlab-cicd)
4. [Jenkins](#jenkins)
5. [Argo Workflows](#argo-workflows)
6. [Azure Pipelines](#azure-pipelines)
7. [CircleCI](#circleci)
8. [Advanced Patterns](#advanced-patterns)

---

## General Integration Principles

### Installation Methods

#### Method 1: Direct Binary Download
```bash
VERSION="0.1.0"
wget https://github.com/user/helmshift/releases/download/v${VERSION}/helmshift-linux-amd64
chmod +x helmshift-linux-amd64
sudo mv helmshift-linux-amd64 /usr/local/bin/helmshift
```

#### Method 2: Go Install
```bash
go install github.com/kdwils/helmshift/cmd/helmshift@latest
```

#### Method 3: Container Image
```dockerfile
FROM alpine:3.18
COPY helmshift /usr/local/bin/
ENTRYPOINT ["helmshift"]
```

### Basic Workflow Pattern

All CI/CD integrations follow this pattern:

```
1. Checkout code
2. Install helmshift
3. Run migration (with -dry-run for validation)
4. Optionally: commit migrated values
5. Optionally: deploy with new values
```

---

## GitHub Actions

### Example 1: Validate Migration on PR

```yaml
# .github/workflows/validate-migration.yml
name: Validate Helm Values Migration

on:
  pull_request:
    paths:
      - 'helm/values/*.yaml'
      - 'helm/migrations/*.yaml'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Download HelmShift
        run: |
          VERSION="0.1.0"
          wget https://github.com/user/helmshift/releases/download/v${VERSION}/helmshift-linux-amd64
          chmod +x helmshift-linux-amd64
          sudo mv helmshift-linux-amd64 /usr/local/bin/helmshift

      - name: Validate migration (dry-run)
        run: |
          helmshift \
            -config helm/migrations/v3-to-v4.yaml \
            -values helm/values/production.yaml \
            -dry-run

      - name: Run migration and save output
        run: |
          helmshift \
            -config helm/migrations/v3-to-v4.yaml \
            -values helm/values/production.yaml \
            -output /tmp/migrated.yaml

      - name: Validate migrated values with Helm
        run: |
          helm template myapp ./charts/myapp \
            -f /tmp/migrated.yaml \
            --validate

      - name: Upload migrated values as artifact
        uses: actions/upload-artifact@v3
        with:
          name: migrated-values
          path: /tmp/migrated.yaml
```

### Example 2: Auto-Migrate and Commit

```yaml
# .github/workflows/auto-migrate.yml
name: Auto-Migrate Helm Values

on:
  workflow_dispatch:
    inputs:
      environment:
        description: 'Environment to migrate'
        required: true
        type: choice
        options:
          - dev
          - staging
          - production

jobs:
  migrate:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      pull-requests: write

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          ref: main

      - name: Install HelmShift
        run: |
          VERSION="0.1.0"
          wget https://github.com/user/helmshift/releases/download/v${VERSION}/helmshift-linux-amd64
          chmod +x helmshift-linux-amd64
          sudo mv helmshift-linux-amd64 /usr/local/bin/helmshift

      - name: Migrate values
        run: |
          helmshift \
            -config migrations/v3-to-v4.yaml \
            -values environments/${{ inputs.environment }}/values.yaml \
            -output environments/${{ inputs.environment }}/values-new.yaml

      - name: Create Pull Request
        uses: peter-evans/create-pull-request@v5
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          commit-message: "chore: migrate ${{ inputs.environment }} values to v4"
          title: "Migrate ${{ inputs.environment }} Helm values to v4"
          body: |
            Automated migration of Helm values for **${{ inputs.environment }}** environment.

            Migration performed using HelmShift with config: `migrations/v3-to-v4.yaml`

            Please review the changes carefully before merging.
          branch: migrate-${{ inputs.environment }}-v4
          delete-branch: true
```

### Example 3: Multi-Environment Migration

```yaml
# .github/workflows/migrate-all-environments.yml
name: Migrate All Environments

on:
  workflow_dispatch:

jobs:
  migrate:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        environment: [dev, staging, production]

    steps:
      - uses: actions/checkout@v4

      - name: Install HelmShift
        run: |
          VERSION="0.1.0"
          wget https://github.com/user/helmshift/releases/download/v${VERSION}/helmshift-linux-amd64
          chmod +x helmshift-linux-amd64
          sudo mv helmshift-linux-amd64 /usr/local/bin/helmshift

      - name: Migrate ${{ matrix.environment }}
        run: |
          helmshift \
            -config migrations/v3-to-v4.yaml \
            -values environments/${{ matrix.environment }}/values.yaml \
            -output environments/${{ matrix.environment }}/values-migrated.yaml

      - name: Upload artifact
        uses: actions/upload-artifact@v3
        with:
          name: ${{ matrix.environment }}-migrated
          path: environments/${{ matrix.environment }}/values-migrated.yaml
```

---

## GitLab CI/CD

### Example 1: Validation Pipeline

```yaml
# .gitlab-ci.yml
stages:
  - validate
  - migrate
  - deploy

variables:
  HELMSHIFT_VERSION: "0.1.0"

.install_helmshift: &install_helmshift
  - wget https://github.com/user/helmshift/releases/download/v${HELMSHIFT_VERSION}/helmshift-linux-amd64
  - chmod +x helmshift-linux-amd64
  - mv helmshift-linux-amd64 /usr/local/bin/helmshift

validate_migration:
  stage: validate
  image: alpine:3.18
  before_script:
    - apk add --no-cache wget
    - *install_helmshift
  script:
    - helmshift -config migrations/v3-to-v4.yaml -values values.yaml -dry-run
  only:
    - merge_requests
    - main

migrate_dev:
  stage: migrate
  image: alpine:3.18
  before_script:
    - apk add --no-cache wget git
    - *install_helmshift
  script:
    - helmshift -config migrations/v3-to-v4.yaml -values environments/dev/values.yaml -output environments/dev/values-migrated.yaml
  artifacts:
    paths:
      - environments/dev/values-migrated.yaml
    expire_in: 1 week
  only:
    - main

migrate_production:
  stage: migrate
  image: alpine:3.18
  before_script:
    - apk add --no-cache wget
    - *install_helmshift
  script:
    - helmshift -config migrations/v3-to-v4.yaml -values environments/production/values.yaml -output environments/production/values-migrated.yaml
  artifacts:
    paths:
      - environments/production/values-migrated.yaml
  when: manual
  only:
    - main
```

### Example 2: Auto-Commit Pattern

```yaml
# .gitlab-ci.yml
migrate_and_commit:
  stage: migrate
  image: alpine:3.18
  before_script:
    - apk add --no-cache wget git
    - git config user.name "GitLab CI"
    - git config user.email "ci@gitlab.com"
    - *install_helmshift
  script:
    - |
      for env in dev staging production; do
        helmshift \
          -config migrations/v3-to-v4.yaml \
          -values environments/${env}/values.yaml \
          -output environments/${env}/values-new.yaml

        mv environments/${env}/values-new.yaml environments/${env}/values.yaml
      done
    - git add environments/*/values.yaml
    - git commit -m "ci: migrate all environments to chart v4"
    - git push origin HEAD:${CI_COMMIT_REF_NAME}
  only:
    - main
  when: manual
```

---

## Jenkins

### Example 1: Declarative Pipeline

```groovy
// Jenkinsfile
pipeline {
    agent any

    parameters {
        choice(
            name: 'ENVIRONMENT',
            choices: ['dev', 'staging', 'production'],
            description: 'Environment to migrate'
        )
    }

    environment {
        HELMSHIFT_VERSION = '0.1.0'
    }

    stages {
        stage('Install HelmShift') {
            steps {
                sh '''
                    wget https://github.com/user/helmshift/releases/download/v${HELMSHIFT_VERSION}/helmshift-linux-amd64
                    chmod +x helmshift-linux-amd64
                    sudo mv helmshift-linux-amd64 /usr/local/bin/helmshift
                '''
            }
        }

        stage('Validate Migration') {
            steps {
                sh """
                    helmshift \
                        -config migrations/v3-to-v4.yaml \
                        -values environments/${params.ENVIRONMENT}/values.yaml \
                        -dry-run
                """
            }
        }

        stage('Perform Migration') {
            steps {
                sh """
                    helmshift \
                        -config migrations/v3-to-v4.yaml \
                        -values environments/${params.ENVIRONMENT}/values.yaml \
                        -output environments/${params.ENVIRONMENT}/values-migrated.yaml
                """
            }
        }

        stage('Validate with Helm') {
            steps {
                sh """
                    helm template myapp ./charts/myapp \
                        -f environments/${params.ENVIRONMENT}/values-migrated.yaml \
                        --validate
                """
            }
        }

        stage('Archive Results') {
            steps {
                archiveArtifacts artifacts: "environments/${params.ENVIRONMENT}/values-migrated.yaml"
            }
        }
    }

    post {
        success {
            echo "Migration completed successfully for ${params.ENVIRONMENT}"
        }
        failure {
            echo "Migration failed for ${params.ENVIRONMENT}"
        }
    }
}
```

### Example 2: Scripted Pipeline with Approval

```groovy
// Jenkinsfile
node {
    def environments = ['dev', 'staging', 'production']

    stage('Checkout') {
        checkout scm
    }

    stage('Install HelmShift') {
        sh '''
            wget https://github.com/user/helmshift/releases/download/v0.1.0/helmshift-linux-amd64
            chmod +x helmshift-linux-amd64
            sudo mv helmshift-linux-amd64 /usr/local/bin/helmshift
        '''
    }

    environments.each { env ->
        stage("Migrate ${env}") {
            sh """
                helmshift \
                    -config migrations/v3-to-v4.yaml \
                    -values environments/${env}/values.yaml \
                    -output environments/${env}/values-migrated.yaml
            """

            if (env == 'production') {
                input message: "Approve ${env} migration?", ok: "Approve"
            }

            sh """
                mv environments/${env}/values-migrated.yaml environments/${env}/values.yaml
                git add environments/${env}/values.yaml
            """
        }
    }

    stage('Commit Changes') {
        sh '''
            git config user.name "Jenkins"
            git config user.email "jenkins@example.com"
            git commit -m "ci: migrate all environments to chart v4"
            git push origin HEAD:main
        '''
    }
}
```

---

## Argo Workflows

```yaml
# argo-migrate-values.yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: helm-values-migration-
spec:
  entrypoint: migrate-values
  arguments:
    parameters:
      - name: environment
        value: "production"

  templates:
    - name: migrate-values
      inputs:
        parameters:
          - name: environment
      steps:
        - - name: validate
            template: helmshift-validate
            arguments:
              parameters:
                - name: environment
                  value: "{{inputs.parameters.environment}}"

        - - name: migrate
            template: helmshift-migrate
            arguments:
              parameters:
                - name: environment
                  value: "{{inputs.parameters.environment}}"

    - name: helmshift-validate
      inputs:
        parameters:
          - name: environment
      container:
        image: alpine:3.18
        command: [sh, -c]
        args:
          - |
            apk add --no-cache wget
            wget https://github.com/user/helmshift/releases/download/v0.1.0/helmshift-linux-amd64
            chmod +x helmshift-linux-amd64
            mv helmshift-linux-amd64 /usr/local/bin/helmshift

            helmshift \
              -config /workspace/migrations/v3-to-v4.yaml \
              -values /workspace/environments/{{inputs.parameters.environment}}/values.yaml \
              -dry-run
        volumeMounts:
          - name: workspace
            mountPath: /workspace

    - name: helmshift-migrate
      inputs:
        parameters:
          - name: environment
      container:
        image: alpine:3.18
        command: [sh, -c]
        args:
          - |
            apk add --no-cache wget
            wget https://github.com/user/helmshift/releases/download/v0.1.0/helmshift-linux-amd64
            chmod +x helmshift-linux-amd64
            mv helmshift-linux-amd64 /usr/local/bin/helmshift

            helmshift \
              -config /workspace/migrations/v3-to-v4.yaml \
              -values /workspace/environments/{{inputs.parameters.environment}}/values.yaml \
              -output /workspace/environments/{{inputs.parameters.environment}}/values-migrated.yaml
        volumeMounts:
          - name: workspace
            mountPath: /workspace
      outputs:
        artifacts:
          - name: migrated-values
            path: /workspace/environments/{{inputs.parameters.environment}}/values-migrated.yaml

  volumes:
    - name: workspace
      persistentVolumeClaim:
        claimName: workspace-pvc
```

---

## Azure Pipelines

```yaml
# azure-pipelines.yml
trigger:
  branches:
    include:
      - main
  paths:
    include:
      - helm/values/*
      - helm/migrations/*

pool:
  vmImage: 'ubuntu-latest'

variables:
  HELMSHIFT_VERSION: '0.1.0'

stages:
  - stage: Validate
    jobs:
      - job: ValidateMigration
        steps:
          - task: Bash@3
            displayName: 'Install HelmShift'
            inputs:
              targetType: 'inline'
              script: |
                wget https://github.com/user/helmshift/releases/download/v$(HELMSHIFT_VERSION)/helmshift-linux-amd64
                chmod +x helmshift-linux-amd64
                sudo mv helmshift-linux-amd64 /usr/local/bin/helmshift

          - task: Bash@3
            displayName: 'Validate Migration (Dry Run)'
            inputs:
              targetType: 'inline'
              script: |
                helmshift \
                  -config migrations/v3-to-v4.yaml \
                  -values helm/values/production.yaml \
                  -dry-run

  - stage: Migrate
    dependsOn: Validate
    condition: succeeded()
    jobs:
      - deployment: MigrateProduction
        environment: 'production'
        strategy:
          runOnce:
            deploy:
              steps:
                - checkout: self

                - task: Bash@3
                  displayName: 'Install HelmShift'
                  inputs:
                    targetType: 'inline'
                    script: |
                      wget https://github.com/user/helmshift/releases/download/v$(HELMSHIFT_VERSION)/helmshift-linux-amd64
                      chmod +x helmshift-linux-amd64
                      sudo mv helmshift-linux-amd64 /usr/local/bin/helmshift

                - task: Bash@3
                  displayName: 'Migrate Values'
                  inputs:
                    targetType: 'inline'
                    script: |
                      helmshift \
                        -config migrations/v3-to-v4.yaml \
                        -values helm/values/production.yaml \
                        -output helm/values/production-migrated.yaml

                - task: PublishBuildArtifacts@1
                  inputs:
                    pathToPublish: 'helm/values/production-migrated.yaml'
                    artifactName: 'migrated-values'
```

---

## CircleCI

```yaml
# .circleci/config.yml
version: 2.1

executors:
  helmshift-executor:
    docker:
      - image: cimg/base:2023.08
    environment:
      HELMSHIFT_VERSION: "0.1.0"

commands:
  install-helmshift:
    steps:
      - run:
          name: Install HelmShift
          command: |
            wget https://github.com/user/helmshift/releases/download/v${HELMSHIFT_VERSION}/helmshift-linux-amd64
            chmod +x helmshift-linux-amd64
            sudo mv helmshift-linux-amd64 /usr/local/bin/helmshift

jobs:
  validate-migration:
    executor: helmshift-executor
    steps:
      - checkout
      - install-helmshift
      - run:
          name: Validate migration
          command: |
            helmshift \
              -config migrations/v3-to-v4.yaml \
              -values values.yaml \
              -dry-run

  migrate-values:
    executor: helmshift-executor
    parameters:
      environment:
        type: string
    steps:
      - checkout
      - install-helmshift
      - run:
          name: Migrate << parameters.environment >>
          command: |
            helmshift \
              -config migrations/v3-to-v4.yaml \
              -values environments/<< parameters.environment >>/values.yaml \
              -output environments/<< parameters.environment >>/values-migrated.yaml
      - store_artifacts:
          path: environments/<< parameters.environment >>/values-migrated.yaml

workflows:
  migration-workflow:
    jobs:
      - validate-migration

      - migrate-values:
          name: migrate-dev
          environment: dev
          requires:
            - validate-migration

      - migrate-values:
          name: migrate-staging
          environment: staging
          requires:
            - migrate-dev

      - hold-production:
          type: approval
          requires:
            - migrate-staging

      - migrate-values:
          name: migrate-production
          environment: production
          requires:
            - hold-production
```

---

## Advanced Patterns

### Pattern 1: Automated PR Creation

```bash
#!/bin/bash
# migrate-and-pr.sh

set -e

ENVIRONMENT=$1
MIGRATION_CONFIG=$2

# Perform migration
helmshift \
  -config "${MIGRATION_CONFIG}" \
  -values "environments/${ENVIRONMENT}/values.yaml" \
  -output "environments/${ENVIRONMENT}/values-new.yaml"

# Create feature branch
BRANCH="migrate-${ENVIRONMENT}-$(date +%Y%m%d)"
git checkout -b "${BRANCH}"

# Replace old values
mv "environments/${ENVIRONMENT}/values-new.yaml" "environments/${ENVIRONMENT}/values.yaml"
git add "environments/${ENVIRONMENT}/values.yaml"
git commit -m "chore: migrate ${ENVIRONMENT} values to new chart version"

# Push and create PR (GitHub CLI)
git push origin "${BRANCH}"
gh pr create \
  --title "Migrate ${ENVIRONMENT} Helm values" \
  --body "Automated migration using HelmShift" \
  --base main \
  --head "${BRANCH}"
```

### Pattern 2: Validation with Schema

```yaml
# CI step combining migration with schema validation
- name: Migrate and validate
  run: |
    # Migrate
    helmshift \
      -config migrations/v3-to-v4.yaml \
      -values values.yaml \
      -output migrated.yaml

    # Validate against JSON schema
    yq eval -o=json migrated.yaml | \
      ajv validate -s schemas/values-v4.schema.json

    # Validate with Helm
    helm template myapp charts/myapp -f migrated.yaml --validate
```

### Pattern 3: Multi-Step Migration Chain

```bash
#!/bin/bash
# Migrate through multiple versions: v1 -> v2 -> v3 -> v4

INPUT="values-v1.yaml"
MIGRATIONS=("v1-to-v2" "v2-to-v3" "v3-to-v4")

for migration in "${MIGRATIONS[@]}"; do
  OUTPUT="values-${migration#*-to-}.yaml"

  echo "Migrating ${INPUT} -> ${OUTPUT}"
  helmshift \
    -config "migrations/${migration}.yaml" \
    -values "${INPUT}" \
    -output "${OUTPUT}"

  INPUT="${OUTPUT}"
done

echo "Final migrated values: ${OUTPUT}"
```

### Pattern 4: Rollback Support

```yaml
# Store original values before migration
- name: Backup original values
  run: |
    cp values.yaml values.yaml.backup

- name: Migrate
  run: |
    helmshift -config migrations/v3-to-v4.yaml -values values.yaml -output values-new.yaml

- name: Validate migration
  run: |
    helm template myapp charts/myapp -f values-new.yaml --validate

- name: Apply or rollback
  run: |
    if [ $? -eq 0 ]; then
      mv values-new.yaml values.yaml
      echo "Migration successful"
    else
      echo "Migration failed, restoring backup"
      mv values.yaml.backup values.yaml
      exit 1
    fi
```

---

## Best Practices

### 1. Always Use Dry-Run First
```bash
# Validate before actual migration
helmshift -config migration.yaml -values values.yaml -dry-run && \
helmshift -config migration.yaml -values values.yaml -output new-values.yaml
```

### 2. Version Control Everything
- Commit migration configs to repository
- Tag migrations with chart versions
- Track changes in values files

### 3. Test Migrations in Lower Environments
```yaml
# Deploy order: dev -> staging -> (approval) -> production
dev -> staging -> production
```

### 4. Use Artifacts
```yaml
# Save migrated values as CI artifacts for review
artifacts:
  paths:
    - "**/values-migrated.yaml"
  expire_in: 30 days
```

### 5. Implement Approvals for Production
```yaml
# Require manual approval for production migrations
migrate_production:
  when: manual
  only:
    - main
```

---

## Troubleshooting

### Common Issues

**Issue**: Migration fails with "script not found"
```bash
# Solution: Ensure script has correct path and is executable
chmod +x migrations/migrate.sh
```

**Issue**: YAML parsing errors
```bash
# Solution: Validate YAML before migration
yamllint values.yaml
```

**Issue**: Timeout in CI
```bash
# Solution: Increase timeout
helmshift -config migration.yaml -values values.yaml -timeout 60s
```

**Issue**: Binary not found in PATH
```bash
# Solution: Use absolute path or verify installation
which helmshift
./helmshift -version
```

---

## Conclusion

HelmShift integrates seamlessly into any CI/CD platform with minimal configuration. The key principles are:

1. Install the binary (download, package manager, or container)
2. Run dry-run validation
3. Perform migration
4. Validate output
5. Commit or deploy

Choose the pattern that best fits your workflow and adapt the examples to your specific needs.

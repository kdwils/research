# Research Projects

This repository contains various research projects and proof-of-concept implementations exploring solutions to common infrastructure and DevOps challenges.

## 📁 Projects

### [HelmShift](./helmshift/) - Helm Values Migration Tool

**Status**: ✅ Proof-of-Concept Complete

**Problem**: When Helm charts introduce breaking changes between versions, users must manually update their `values.yaml` files—a process that is error-prone, time-consuming, and difficult to automate in CI/CD pipelines.

**Solution**: HelmShift is a flexible CLI tool that automates Helm values migration using user-defined patches in multiple formats.

**Key Features**:
- 🔄 **Multi-format patch support**: JSON Patch (RFC 6902), yq expressions, and executable scripts
- 📝 **User-authored migrations**: Patches stored separately from tool for maximum flexibility
- 🔧 **CI/CD native**: Dry-run mode, validation, and platform-agnostic design
- ✅ **Safety first**: Pre/post-migration validation with custom rules
- 🚀 **Standalone**: No dependencies on Helm or Kubernetes

**Technology**: Go, JSON Patch (RFC 6902), YAML processing

**Documentation**:
- [Quick Start Guide](./helmshift/README.md)
- [Research & Analysis](./helmshift/docs/research.md) - Comprehensive analysis of existing tools
- [CI/CD Integration](./helmshift/docs/ci-integration.md) - Platform-specific patterns
- [Patch Formats](./helmshift/docs/patch-formats.md) - Complete specifications

**Example Use Case**:
```bash
# Migrate nginx-ingress chart from v3 to v4
cd helmshift
go build -o helmshift ./cmd/helmshift

./helmshift \
  -config examples/patches/nginx-ingress/jsonpatch-migration.yaml \
  -values examples/values/nginx-ingress/v3-values.yaml \
  -output v4-values.yaml
```

---

## 🎯 Project Goals

Each research project in this repository aims to:

1. **Identify a real problem** - Address genuine pain points in DevOps/infrastructure workflows
2. **Research existing solutions** - Analyze what's available and identify gaps
3. **Design a better approach** - Propose improvements or novel solutions
4. **Implement a PoC** - Build working proof-of-concept demonstrating viability
5. **Document thoroughly** - Provide comprehensive documentation for evaluation and reproduction

## 📊 Project Structure

Each project follows a consistent structure:

```
project-name/
├── README.md                   # Project overview and quick start
├── docs/
│   ├── research.md            # Research findings and analysis
│   ├── architecture.md        # Design and architecture (if applicable)
│   └── [additional docs]
├── cmd/                       # CLI tools (if applicable)
├── pkg/                       # Go packages (if applicable)
├── examples/                  # Usage examples and demos
└── [project-specific files]
```

## 🚀 Getting Started

To explore a specific project:

1. Navigate to the project directory
2. Read the project's README.md for overview and quick start
3. Check the `docs/` folder for detailed documentation
4. Try the examples in `examples/` (if available)

## 🔬 Research Methodology

Projects follow this research process:

1. **Problem Definition** - Clearly articulate the problem and requirements
2. **Existing Solution Analysis** - Survey and evaluate current tools/approaches
3. **Gap Analysis** - Identify shortcomings and opportunities for improvement
4. **Solution Design** - Architect a better approach with clear trade-offs
5. **Proof-of-Concept** - Implement core functionality to validate feasibility
6. **Documentation** - Write comprehensive docs for evaluation and future development

## 📝 Documentation Standards

All projects include:

- **README.md** - Quick start, features, and basic usage
- **docs/research.md** - Problem analysis, existing solutions, and proposed approach
- **Code comments** - Well-commented implementation
- **Examples** - Working examples demonstrating key features
- **CI/CD patterns** - Integration guides for automation (where applicable)

## 🛠️ Technology Stack

Projects typically use:

- **Languages**: Go (primary), Python, Bash
- **Tooling**: Docker, Kubernetes, Helm, CI/CD platforms
- **Principles**: Cloud-native, 12-factor, GitOps

## 📈 Project Status Legend

- ✅ **Complete** - Fully implemented PoC with comprehensive documentation
- 🚧 **In Progress** - Active development
- 💡 **Planned** - Identified for future research
- 🔍 **Exploratory** - Initial research phase

## 🤝 Contributing

These are research projects for exploration and learning. If you'd like to:

- **Suggest improvements** - Open an issue with ideas
- **Report issues** - File bugs or documentation gaps
- **Extend research** - Fork and build upon the work

Please note: These are proof-of-concept implementations meant for evaluation and learning, not production use without further development.

## 📄 License

Each project may have its own license. Check individual project directories for details.

Default: Apache 2.0 (unless otherwise specified)

---

## 📚 Additional Resources

### General DevOps & Infrastructure

- [The Twelve-Factor App](https://12factor.net/)
- [GitOps Principles](https://opengitops.dev/)
- [CNCF Landscape](https://landscape.cncf.io/)

### Kubernetes & Helm

- [Helm Documentation](https://helm.sh/docs/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)

### Standards & RFCs

- [RFC 6902: JSON Patch](https://datatracker.ietf.org/doc/html/rfc6902)
- [RFC 7396: JSON Merge Patch](https://datatracker.ietf.org/doc/html/rfc7396)
- [YAML Specification](https://yaml.org/spec/)

---

**Made with ❤️ for the DevOps community**

*Exploring solutions, one research project at a time.*

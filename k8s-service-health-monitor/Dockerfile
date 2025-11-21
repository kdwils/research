# Build the controller and dashboard binaries
FROM golang:1.22 as builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Cache deps before building and copying source
RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o controller cmd/controller/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o dashboard cmd/dashboard/main.go

# Use distroless as minimal base image
FROM gcr.io/distroless/static:nonroot

WORKDIR /

# Copy binaries from builder
COPY --from=builder /workspace/controller .
COPY --from=builder /workspace/dashboard .

USER 65532:65532

# Default to controller, but can be overridden
ENTRYPOINT ["/controller"]

# Stage 1 — Build the Go binary
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY web/ web/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/serviceaccounts ./cmd/serviceaccounts

# Stage 2 — Download and verify uctl; pre-create ~/.uctl so uctl doesn't try to
# create it at runtime (which would fail under a read-only root filesystem).
FROM alpine:3.21 AS uctl-fetch
ARG UCTL_VERSION=v0.1.20
ARG UCTL_SHA256=6bc5bc36d419bc464fa7827a5d8e820d1d8db79ba00cd365311eb4a4d0839d68
RUN apk add --no-cache curl && \
    curl -fsSL \
      "https://github.com/unionai/uctl/releases/download/${UCTL_VERSION}/uctl_Linux_x86_64.tar.gz" \
      -o /tmp/uctl.tar.gz && \
    echo "${UCTL_SHA256}  /tmp/uctl.tar.gz" | sha256sum -c - && \
    tar -xzf /tmp/uctl.tar.gz -C /tmp && \
    chmod +x /tmp/uctl && \
    mkdir -p /home/nonroot/.uctl && \
    touch /home/nonroot/.uctl/.init

# Stage 3 — Minimal runtime image with glibc (for uctl compatibility)
# Uses distroless/cc which includes glibc/libstdc++ but no shell or package manager.
# The :nonroot variant runs as UID 65532 by default.
FROM gcr.io/distroless/cc-debian12:nonroot
# HOME must be set explicitly — distroless does not set it, causing uctl to
# resolve ~ as / and attempt to create /.uctl on a read-only filesystem.
ENV HOME=/home/nonroot
COPY --from=builder /bin/serviceaccounts /bin/serviceaccounts
COPY --from=uctl-fetch /tmp/uctl /usr/local/bin/uctl
# Pre-create ~/.uctl so it exists in the image layer. In Kubernetes the directory
# is shadowed by an emptyDir volume mount so uctl can write to it at runtime.
COPY --from=uctl-fetch --chown=65532:65532 /home/nonroot/.uctl /home/nonroot/.uctl
EXPOSE 8080
ENTRYPOINT ["/bin/serviceaccounts"]

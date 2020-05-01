# Copyright 2020 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Build the manager binary
FROM golang:1.13.8 as builder
WORKDIR /workspace

# Run this with docker build --build_arg goproxy=$(go env GOPROXY) to override the goproxy
ARG goproxy=https://proxy.golang.org
# Run this with docker build --build_arg package=./controlplane/kubeadm or --build_arg package=./bootstrap/kubeadm
ENV GOPROXY=$goproxy

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the sources
COPY ./ ./

# Cache the go build
RUN go build .

# Build
ARG package=.
ARG ARCH

# Do not force rebuild of up-to-date packages (do not use -a)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} \
    go build -ldflags '-extldflags "-static"' \
    -o go-runner ${package}

# Production image
FROM gcr.io/distroless/static:latest
LABEL maintainers="Kubernetes Authors"
LABEL description="go based runner for distroless scenarios"
WORKDIR /
COPY --from=builder /workspace/go-runner .
USER nobody
ENTRYPOINT ["/go-runner"]

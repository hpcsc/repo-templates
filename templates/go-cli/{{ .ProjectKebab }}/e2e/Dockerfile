FROM golang:{{.Scaffold.GoVersion}} AS builder
# go.mod can ask for a newer Go patch release than the image has
ENV GOTOOLCHAIN=auto
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG BUILD_TAG=v0.1.0
RUN CGO_ENABLED=0 go build -ldflags "-X github.com/hpcsc/{{.ProjectKebab}}/internal/version.releaseTag=${BUILD_TAG}" \
    -o /{{.ProjectKebab}} ./cmd/{{.ProjectKebab}}

FROM node:24-trixie-slim
# python3, make and g++ build node-pty when no prebuilt node-pty fits the platform
RUN apt-get update \
    && apt-get install -y --no-install-recommends python3 make g++ \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /e2e
COPY e2e/package.json e2e/package-lock.json* ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install --no-audit --no-fund; fi
COPY e2e ./
COPY --from=builder /{{.ProjectKebab}} /usr/local/bin/{{.ProjectKebab}}
ARG BUILD_TAG=v0.1.0
ENV EXECUTABLE=/usr/local/bin/{{.ProjectKebab}} BUILD_TAG=${BUILD_TAG}
CMD ["npx", "vitest", "run"]

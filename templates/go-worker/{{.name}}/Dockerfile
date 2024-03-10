FROM golang:{{.go_version}}-alpine AS build-env
WORKDIR /build

ARG VERSION=master
ADD . .
RUN GOARCH=$TARGETARCH \
    GOOS=linux \
    CGO_ENABLED=0 \
      go build --ldflags "-extldflags -static -X 'main.Version=${VERSION}'" \
        -o {{.name}} \
        ./cmd/{{.name}}/main.go

FROM ubuntu:22.04
WORKDIR /app

COPY --from=build-env /build/{{.name}} .

ENTRYPOINT ["/app/{{.name}}"]

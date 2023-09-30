FROM golang:{{cookiecutter.go_version}}-alpine AS build-env
WORKDIR /build

ARG VERSION=master
ADD . .
RUN GOARCH=$TARGETARCH \
    GOOS=linux \
    CGO_ENABLED=0 \
      go build --ldflags "-extldflags -static -X 'main.Version=${VERSION}'" \
        -o {{cookiecutter.name}} \
        ./cmd/{{cookiecutter.name}}/main.go

FROM ubuntu:22.04
WORKDIR /app

COPY --from=build-env /build/{{cookiecutter.name}} .

ENTRYPOINT ["/app/{{cookiecutter.name}}"]

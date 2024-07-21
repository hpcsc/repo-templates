FROM golang:{{.go_version}}-alpine AS build-env
WORKDIR /build

ARG VERSION=master
ADD . .
RUN GOARCH=$TARGETARCH \
    GOOS=linux \
    CGO_ENABLED=0 \
      go build --ldflags "-extldflags -static -X 'github.com/hpcsc/{{.name}}/internal/usecase/root.Version=${VERSION}'" \
        -o {{.name}} \
        ./cmd/{{.name}}/main.go

FROM scratch
WORKDIR /app

COPY --from=build-env /build/{{.name}} .

ENTRYPOINT ["/app/{{.name}}"]

FROM golang:{{.Scaffold.GoVersion}}-alpine AS build-env
WORKDIR /build

ARG VERSION=master
ADD . .
RUN GOARCH=$TARGETARCH \
    GOOS=linux \
    CGO_ENABLED=0 \
      go build --ldflags "-extldflags -static -X 'main.Version=${VERSION}'" \
        -o {{.ProjectKebab}} \
        ./cmd/{{.ProjectKebab}}/main.go

FROM scratch
WORKDIR /app

COPY --from=build-env /build/{{.ProjectKebab}} .

ENTRYPOINT ["/app/{{.ProjectKebab}}"]

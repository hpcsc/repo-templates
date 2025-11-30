FROM golang:{{.Scaffold.GoVersion}}-trixie
WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  bats \
  expect \
  && rm -rf /var/lib/apt/lists/*

RUN /bin/bash -c "echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | tee /etc/apt/sources.list.d/goreleaser.list"

RUN apt-get update && apt-get install -y --no-install-recommends \
  goreleaser \
  && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

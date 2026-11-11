#!/bin/bash

set -euo pipefail

export COMPOSE_PROJECT_NAME={{ .ProjectKebab }}-integration-test
export COMPOSE_FILE=./docker-compose.test.yml

docker compose down --remove-orphans
{{- if .Scaffold.PushNATS }}
docker compose up --wait db nats
{{- else }}
docker compose up --wait db
{{- end }}
docker compose up --exit-code-from migrate migrate

#!/bin/bash

set -euo pipefail

export COMPOSE_PROJECT_NAME={{ .ProjectKebab }}-integration-test
export COMPOSE_FILE=./docker-compose.test.yml

docker compose down --remove-orphans
docker compose up --wait db
docker compose up --exit-code-from migrate migrate

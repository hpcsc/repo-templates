#!/bin/bash

set -euo pipefail

GOFLAGS=-buildvcs=false goreleaser build --clean --single-target --snapshot -o ${EXECUTABLE}

bats /app/e2e

rm -rvf $(dirname ${EXECUTABLE})

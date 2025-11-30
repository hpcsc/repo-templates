#!/bin/bash

set -euo pipefail

BUILD=${BUILD:-false}

if [ "${BUILD}" = "true" ] || [ ! -f ${EXECUTABLE} ]; then
  GOFLAGS=-buildvcs=false goreleaser build --clean --single-target --snapshot -o ${EXECUTABLE}
fi

bats /app/e2e

rm -rvf $(dirname ${EXECUTABLE})

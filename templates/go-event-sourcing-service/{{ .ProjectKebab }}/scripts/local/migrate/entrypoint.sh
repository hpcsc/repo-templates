#!/bin/sh

COMMAND="${@:-up}"

TIMEOUT_PERIOD=60
DELAY=2

wait_for() {
    set +e
    local cmd="$@"
    local t=$TIMEOUT_PERIOD

    eval "${cmd}"
    until [ $? = 0 ]  ; do
        t=$((t - DELAY))
        if [[ $t -eq 0 ]]; then
            echo "=== ${DB_HOST}:${DB_PORT} is not up after $TIMEOUT_PERIOD seconds"
            set -e
            exit 1
        fi

        echo "=== ${DB_HOST}:${DB_PORT} is not up yet, remaining time: $t seconds"
        sleep $DELAY
        eval "${cmd}"
    done

    echo "=== ${DB_HOST}:${DB_PORT} is up"
    set -e
}

wait_for nc -z "${DB_HOST}" "${DB_PORT}"
migrate \
  -database "postgres://${DB_USERNAME}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable" \
  -path /app/migrations \
  "${COMMAND}"

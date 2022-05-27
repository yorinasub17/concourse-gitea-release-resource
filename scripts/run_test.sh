#!/bin/bash

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly TEST_DIR="$SCRIPT_DIR"/../test

>&2 echo 'Building test gitea container image'
IMAGE_ID="$(docker build -q "$TEST_DIR"/env)"
>&2 echo 'Starting test gitea container'
CONTAINER_ID="$(docker run --rm -d -p 3000:3000 "$IMAGE_ID")"

>&2 echo "Sleeping for 10 seconds to give container $CONTAINER_ID a moment to boot"
sleep 10
>&2 echo 'Loading test repo and releases to test gitea container'
go run "$TEST_DIR"/setup

go test -v -count 1 ./...
TEST_EXIT_CODE="$?"

>&2 echo 'Stopping test gitea container'
docker stop "$CONTAINER_ID"
sleep 10
echo "Sleeping for 10 seconds to give container $CONTAINER_ID a moment to stop"

>&2 echo 'Removing test gitea container image'
docker rmi "$IMAGE_ID"

exit "$TEST_EXIT_CODE"

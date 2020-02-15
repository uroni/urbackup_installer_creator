#!/bin/bash

set -e

CNAME="installer-creator1"

docker stop "$CNAME" || true
docker rm "$CNAME" || true

docker run -d -p 5000:5000 --name "$CNAME" installer_creator

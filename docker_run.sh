#!/bin/bash

set -e

CNAME="installer-creator1"

docker stop "$CNAME" || true
docker rm "$CNAME" || true

docker run -d -p 127.0.0.1:5000:5000 --name "$CNAME" installer_creator

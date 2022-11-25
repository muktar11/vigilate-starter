#!/bin/zsh

# This is the bare minimum to run in development. For full list of flags,
# run ./vigilate -help

go build -o vigilatestarter cmd/web/*.go && ./vigilatestarter \
-dbuser='root' \
-pusherHost='localhost' \
-pusherKey='abc123' \
-pusherSecret='123abc' \
-pusherApp="1" \
-pusherPort="8080" \
-pusherSecure=false
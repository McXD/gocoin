#!/usr/bin/env bash

export PORT=8083
go run ../cmd/node -c --p2p-port=8846 --root="/tmp/gocoin2" --seed="/ip4/127.0.0.1/tcp/8844/p2p/QmWfDG7hW4393BW5QmKfrdwHSK4umj5tGE3c6ibgHTbmCA"
#!/usr/bin/env bash

export PORT=8082
go run ../cmd/node -c --p2p-port=8845 --root="/tmp/gocoin1" --seed="/ip4/127.0.0.1/tcp/8844/p2p/QmWfDG7hW4393BW5QmKfrdwHSK4umj5tGE3c6ibgHTbmCA"
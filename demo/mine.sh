#!/usr/bin/env bash

export PORT=8081
go run ../cmd/node --mine=true --clean=true --p2p-port=8844 --rand-seed=223344
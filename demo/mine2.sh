#!/usr/bin/env bash

export PORT=8083
go run ../cmd/node --mine=true --clean=true --p2p-port=8846 --seed="/ip4/127.0.0.1/tcp/8844/p2p/QmWHgVGPuFgwXeY3JWdYukzAr7hVC7hSLMjpSGLiip5677"
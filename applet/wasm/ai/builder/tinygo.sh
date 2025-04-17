#!/bin/bash

go mod tidy
tinygo build -o main.wasm -target wasi ./src/

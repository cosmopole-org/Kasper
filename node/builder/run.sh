#!/bin/bash

rm -r ../src/shell/api/pluggers
rm -r ../src/shell/api/main

rm -r ../src/shell/machiner/pluggers
rm -r ../src/shell/machiner/main

go run ./pluggergen.go "../src/shell/api" "../src/shell/machiner"

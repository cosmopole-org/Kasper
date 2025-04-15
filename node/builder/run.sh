#!/bin/bash

rm -r ../src/shell/api/pluggers
rm -r ../src/shell/api/main

rm -r ../src/shell/machiner/pluggers
rm -r ../src/shell/machiner/main

rm -r ../src/plugins/social/pluggers
rm -r ../src/plugins/social/main

go run ./pluggergen.go "../src/shell/api" "../src/shell/machiner" "../src/plugins/social"

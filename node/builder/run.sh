#!/bin/bash

rm -r ../src/shell/api/pluggers
rm -r ../src/shell/api/main

rm -r ../src/shell/machiner/pluggers
rm -r ../src/shell/machiner/main

rm -r ../src/plugins/admin/pluggers
rm -r ../src/plugins/admin/main

rm -r ../src/plugins/social/pluggers
rm -r ../src/plugins/social/main

rm -r ../src/plugins/game/pluggers
rm -r ../src/plugins/game/main

go run ./pluggergen.go "../src/shell/api" "../src/shell/machiner" "../src/plugins/admin" "../src/plugins/social" "../src/plugins/game"

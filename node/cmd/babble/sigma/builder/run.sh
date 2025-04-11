#!/bin/bash

rm -r ../api/pluggers
rm -r ../api/main

rm -r ../machiner/pluggers
rm -r ../machiner/main

rm -r ../plugins/admin/pluggers
rm -r ../plugins/admin/main

rm -r ../plugins/social/pluggers
rm -r ../plugins/social/main

rm -r ../plugins/game/pluggers
rm -r ../plugins/game/main

go run ./pluggergen.go "../api" "../machiner" "../plugins/admin" "../plugins/social" "../plugins/game"

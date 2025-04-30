#!/bin/bash

token=$1
machineId=$2

rm $(pwd)/../temp.txt
cp $(pwd)/../Dockerfile $(pwd)/Dockerfile

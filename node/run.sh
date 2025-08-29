#!/bin/bash

echo 'nameserver 10.10.0.4:5353' > /etc/resolv.conf

bash run-fcvmm.sh

./kasper

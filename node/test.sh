#!/bin/bash

latest=$(wget "http://spec.ccfc.min.s3.amazonaws.com/?prefix=firecracker-ci/v1.9/aarch64/vmlinux-5.10&list-type=2" -O - 2>/dev/null | grep "(?<=<Key>)(firecracker-ci/v1.9/aarch64/vmlinux-5\.10\.[0-9]{3})(?=</Key>)" -o -P)
kernel_url=https://s3.amazonaws.com/spec.ccfc.min/${latest}
echo ${kernel_url}
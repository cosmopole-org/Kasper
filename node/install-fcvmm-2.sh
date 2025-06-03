#!/bin/bash

# Mount and install Ubuntu
mkdir -p /mnt/rootfs
mount -o loop rootfs/rootfs.ext4 /mnt/rootfs
debootstrap --arch=amd64 focal /mnt/rootfs
umount /mnt/rootfs

# Set up networking
ip link add name br0 type bridge
ip addr add 172.16.0.1/24 dev br0
ip link set br0 up

iptables -t nat -A POSTROUTING -o $(ip route get 1 | awk '{print $5}') -j MASQUERADE
iptables -A FORWARD -i br0 -o $(ip route get 1 | awk '{print $5}') -j ACCEPT
sysctl -w net.ipv4.ip_forward=1

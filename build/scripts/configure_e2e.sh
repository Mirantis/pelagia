#!/bin/sh

set -xe

mkdir -p /root/.ssh

cat <<EOF > /root/.ssh/config
ControlMaster     auto
ControlPath       /tmp/.ssh_control-%C
ControlPersist    yes
Host *
    Compression=yes
    ServerAliveInterval=10
    ServerAliveCountMax=3
    AddressFamily=inet
    CheckHostIP=no
    UserKnownHostsFile=/dev/null
    LogLevel=ERROR
    StrictHostKeyChecking=no
EOF

chown 0:0 /root/.ssh/config
# use pre-defined test ssh key
[ -f /root/ssh_key ] && mv /root/ssh_key /root/.ssh/id_rsa
[ -f /root/.ssh/id_rsa ] && chown 0:0 /root/.ssh/id_rsa

#!/bin/bash
set -ex -o pipefail

# Install NSQ 0.3.8
cd /tmp
curl -L https://s3.amazonaws.com/bitly-downloads/nsq/nsq-0.3.8.linux-amd64.go1.6.2.tar.gz -o nsq-0.3.8.linux-amd64.go1.6.2.tar.gz
tar -zxvf nsq-0.3.8.linux-amd64.go1.6.2.tar.gz
sudo cp -R nsq-0.3.8.linux-amd64.go1.6.2/bin/. /usr/local/bin

# Start nsqlookupd & nsqd
nsqlookupd &
nsqd -lookupd-tcp-address localhost:4160 &

# Emit test message
echo 'test' | to_nsq -nsqd-tcp-address localhost:4150 -topic test

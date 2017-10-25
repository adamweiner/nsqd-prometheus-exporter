#!/bin/bash
set -ex -o pipefail

# Install NSQ 1.0.0
cd /tmp
curl -L https://s3.amazonaws.com/bitly-downloads/nsq/nsq-1.0.0-compat.linux-amd64.go1.8.tar.gz -o nsq-1.0.0-compat.linux-amd64.go1.8.tar.gz
tar -zxvf nsq-1.0.0-compat.linux-amd64.go1.8.tar.gz
sudo cp -R nsq-1.0.0-compat.linux-amd64.go1.8/bin/. /usr/local/bin

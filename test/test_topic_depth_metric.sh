#!/bin/bash
set -ex -o pipefail

# Emit test message
echo 'test' | to_nsq -nsqd-tcp-address localhost:4150 -topic test -rate 2

# Test for expected metric
curl -s localhost:30000/metrics > metrics.out
cat metrics.out
cat metrics.out | grep 'nsqd_depth{channel="",paused="false",topic="test",type="topic"} 1'

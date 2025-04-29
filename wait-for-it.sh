#!/usr/bin/env bash

HOST="$1"
PORT="$2"
TIMEOUT="${3:-30}"

echo "Waiting for $HOST:$PORT for up to $TIMEOUT seconds..."

start_ts=$(date +%s)

while :
do
    (echo > /dev/tcp/$HOST/$PORT) >/dev/null 2>&1
    result=$?
    
    if [ $result -eq 0 ]; then
        end_ts=$(date +%s)
        echo "Host $HOST:$PORT is available after $((end_ts - start_ts)) seconds."
        exit 0
    fi

    now_ts=$(date +%s)
    if [ $((now_ts - start_ts)) -ge $TIMEOUT ]; then
        echo "Timeout waiting for $HOST:$PORT after $TIMEOUT seconds."
        exit 1
    fi

    sleep 1
done
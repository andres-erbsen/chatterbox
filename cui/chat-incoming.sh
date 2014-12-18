#!/bin/bash

while true; do
	head -99999999 "$1/conversations"/*/*-*-*T*:*:*Z-*
	inotifywait -q -e create "$1"/conversations/{*,} > /dev/null
done

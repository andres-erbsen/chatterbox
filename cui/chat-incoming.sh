#!/bin/sh

cd "$ROOTDIR/conversations/$CONV"

while true; do
	clear
	head -99999999 -v *-*-*T*:*:*Z-*
	inotifywait -q -e create -e moved_to * . > /dev/null
done

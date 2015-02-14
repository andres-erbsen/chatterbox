#!/bin/sh

if [ "$#" -ne 2 ]; then
     echo "Usage: $0 <ROOTDIR> <CONV>" >&2
	 exit 2
fi

ROOTDIR="$1"
CONV="$2"

cd "$ROOTDIR/conversations/$CONV"

while true; do
	clear
	head -99999999 -v $(ls | grep -v metadata.pb)
	inotifywait -q -e create -e moved_to * . > /dev/null
done

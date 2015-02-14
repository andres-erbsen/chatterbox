#!/bin/sh

if [ "$#" -ne 2 ]; then
     echo "Usage: $0 <ROOTDIR> <CONV>" >&2
	 exit 2
fi

ROOTDIR="$1"
CONV="$2"

if [[ ! -x "$(which $EDITOR)" ]]; then
	EDITOR=vim
fi

while true; do
	clear
	file=$(mktemp "$ROOTDIR/tmp/cui-message-being-edited.$$.XXXXXXXXXX")
	$EDITOR "$file"
	grep . "$file" > /dev/null || break
	mv "$file" "$ROOTDIR/outbox/$CONV"
done

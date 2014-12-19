#!/bin/sh

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

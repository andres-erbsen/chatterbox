#!/bin/sh

ROOT="$1"

inotifywait -q -r -m -e moved_to --format "%w%f" ~/.chatterbox/andres/conversations/ | while read -r path; do
	message=$(cat "$path")
	sender=$(basename "$path" | cut -d '-' -f4)
	dir=$(dirname "$path")
	dirname=$(basename "$dir")
	conv=$(echo $dirname | sed 's: %between.*::g')
	notify-send "$conv: $sender" "$message"
done

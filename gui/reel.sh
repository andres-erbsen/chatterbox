#!/bin/sh

set -euo pipefail
IFS=$'\n\t'

while true; do
	./gui &
	pid="$!"
	case "$(inotifywait -q -e modify conversation.go conversation.qml | cut -d' ' -f1)" in
	  "conversation.go") go build
		 ;;
	esac
	kill -TERM "$pid"
done

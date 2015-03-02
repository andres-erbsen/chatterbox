#!/bin/sh

set -euo pipefail
IFS=$'\n\t'

while true; do
	env QRC_REPACK=1 ./gui &
	pid="$!"
	case "$(inotifywait -q -e modify conversation.go qml/conversation.qml | cut -d' ' -f1)" in
	  "conversation.go") go build
		 ;;
	esac
	kill -TERM "$pid"
done

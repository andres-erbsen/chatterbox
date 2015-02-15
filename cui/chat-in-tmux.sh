#!/bin/bash
set -euo pipefail
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

ROOT="$(echo "$1" | awk -F'conversations' '{print($1)}')"
CONV="$(echo "$1" | awk -F'conversations' '{print($2)}')"

if [ ! -d "$1" ]; then
	echo "run chat-create to initialize a conversation:"
	chat-create -help
	exit 1
fi 

tmux split-window -v -l 3
tmux send-keys -t 1 "\"$DIR/chat-incoming.sh\" \"$ROOT\" \"$CONV\"" enter
tmux swap-pane -D -t 1
tmux select-pane -t 1
"$DIR/chat-outgoing.sh" "$ROOT" "$CONV"
tmux kill-pane -t 0

#!/bin/bash
set -euo pipefail
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

tmux split-window -v -l 3
tmux send-keys -t 1 "\"$DIR/chat-incoming.sh\" \"$1\" \"$2\"" enter
tmux swap-pane -D -t 1
tmux select-pane -t 1
"$DIR/chat-outgoing.sh" "$1" "$2"
tmux kill-pane -t 0

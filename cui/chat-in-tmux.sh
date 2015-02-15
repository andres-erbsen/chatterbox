#!/bin/sh

tmux split-window -v -l 3
tmux send-keys -t 1 "./chat-incoming.sh \"$1\" \"$2\"" enter
tmux swap-pane -D -t 1
tmux select-pane -t 1
./chat-outgoing.sh "$1" "$2"
tmux kill-pane -t 0

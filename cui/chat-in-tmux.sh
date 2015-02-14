#!/bin/sh

tmux split-window -v -l 3
tmux send-keys -t 1 "env \"EDITOR=$EDITOR\" ./chat-outgoing.sh \"$1\" \"$2\"" enter
tmux select-pane -t 1
exec ./chat-incoming.sh "$1" "$2"

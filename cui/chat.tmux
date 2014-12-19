#!/usr/bin/tmux source-file

set-option -ga update-environment ' ROOTDIR CONV EDITOR'

new-session -d
split-window -d -t 0 -v -l 3
send-keys -t 0 './chat-incoming.sh' enter
send-keys -t 1 './chat-outgoing.sh' enter
select-pane -t 1
attach

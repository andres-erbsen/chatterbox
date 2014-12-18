#!/usr/bin/tmux source-file

new-session -d
split-window -d -t 0 -v
send-keys -t 0 './chat-incoming.sh /home/andres/.chatterbox/andreser' enter
send-keys -t 1 './chat-outgoing.sh /home/andres/.chatterbox/andreser andres' enter
select-pane -t 1
attach

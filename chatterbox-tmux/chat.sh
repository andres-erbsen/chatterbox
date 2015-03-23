#!/bin/sh

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

if [ "$#" -ne 2 ]; then
     echo "Usage: $0 <ROOTDIR> <CONV>" >&2
	 exit 2
fi

if [[ -z "$TMUX" ]]; then
	tmux new "$DIR/chat-in-tmux.sh $1 $2"
else
	exec "$DIR/chat-in-tmux.sh" $1 $2
fi

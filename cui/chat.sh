#!/bin/sh

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

env "ROOTDIR=$1" "CONV=$2" tmux source-file $(realpath "$DIR/chat.tmux")

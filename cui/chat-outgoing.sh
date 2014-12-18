#!/bin/bash

while true; do
	read msg
	send_message "$1" "$2" "chat" "$msg"
done

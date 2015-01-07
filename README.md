Account creation: `$ROOT` is the directory where you want to set up your Chatterbox files, `$DENAME` is your dename username, `$SERVER_ADDR` is the address of your home server, and `$SERVER_PUBKEY` is that server's public key. It will output a dename command for you to run to  upload your Chatterbox profile.


	# step 1: get a dename account. See <dename.mit.edu>.
	mkdir $ROOT
	`chatterbox-init -account-directory=$ROOT -dename=$DENAME -server-host=$SERVER_ADDR -server-pubkey=$SERVER_PUBKEY`

Example :

	# ./client/chatterbox-init/chatterbox-init  -account-directory=. -dename=andres -server-host=chatterbox.xvm.mit.edu -server-pubkey=70eb7fb3e6c85c006ba76b48208ccf75f99eb6ec98dffb4292388369cb197e30

Unfortunately, the said server does not work currently (6am Sun) because server version  is behind and I am failing to log in `xvm-console`...

Starting a server: 

	mkdir serverdir
	./server/keygen/keygen > pkfile 2> skfile
	./server/server/server skfile pkfile serverdir :1984
	xxd -p pkfile # server's public key

Receiving messages:

	./client/chatterboxd/chatterboxd $ROOT

Sending a message:

	./client/send_message/send_message $ROOT to_dename subject message...text

Inspecting received messages:

	tree $ROOT

Example:

	$ROOT
	├── [-rw-------]  config.pb
	├── [drwx------]  conversations
	│   └── [drwx------]  2014-12-12T09:46:14Z-david-davidtest
	│       ├── [-rw-------]  2014-12-12T09:46:14Z-david
	│       ├── [-rw-------]  2014-12-12T09:47:54Z-davidtest
	│       └── [-rw-------]  metadata.pb
	├── [drwx------]  journal
	├── [drwx------]  keys
	│   ├── [-rw-------]  prekeys.pb
	│   └── [drwx------]  ratchet
	│       └── [-rw-------]  davidtest
	├── [drwx------]  outbox
	│   └── [drwx------]  2014-12-12T09:46:14Z-david-davidtest
	│       └── [-rw-------]  metadata.pb
	├── [-rw-------]  profile.pb
	├── [drwx------]  tmp
	└── [drwx------]  ui_info

The `conversations` folder is both "inbox" and "sent", and has one subfolder per
thread. New messages can be sent by dropping them into the corresponding folder
in outbox.

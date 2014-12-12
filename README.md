Account creation: 

	# step 1: get a dename account. See <dename.mit.edu>.
	mkdir userdir
	./client/chatterbox-init/chatterbox-init  -account-directory=. -dename=andres -server-host=chatterbox.xvm.mit.edu -server-pubkey=70eb7fb3e6c85c006ba76b48208ccf75f99eb6ec98dffb4292388369cb197e30
	# and follow the instructions
	# Unfortunately, the said server does not work currently (6am Sun) because server version
      is behind and I am failing to log in `xvm-console`...

Starting a server: 

	mkdir serverdir
	./server/keygen/keygen > pkfile 2> skfile
	./server/server/server skfile pkfile serverdir :1984
	xxd -p pkfile # server's public key

Receiving messages:

	./client/chatterboxd/chatterboxd userdir

Sending a message:

	./client/send_message/send_message userdir to_dename subject message...text

Inspecting received messages:

	tree userdir

Example:

	userdir
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

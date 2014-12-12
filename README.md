
## Quick start

# Server

# Client
`chatterbox-init -account-directory=$ROOT -dename=$DENAME -server-host=$SERVER_ADDR -server-pubkey=$SERVER_PUBKEY`

where `$ROOT` is the directory where you want to set up your Chatterbox files, `$DENAME` is your dename username, `$SERVER_ADDR` is the address of your home server, and `$SERVER_PUBKEY` is that server's public key. It will output a dename command for you to run to  upload your Chatterbox profile.

`chatterboxd $ROOT`

launches the daemon. You can now send and receive messages.


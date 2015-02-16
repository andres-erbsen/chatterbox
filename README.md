1. Install [`dename`](https://github.com/andres-erbsen/dename) and [get an account](https://dename.mit.edu/).

2. Download, compile, install

		go get github.com/andres-erbsen/chatterbox/client/{chatterboxd,chatterbox-init,chat-create}

3. Create an account:

		chatterbox-init  -account-directory=/home/${USER}/.chatterbox/${DENAME_USER} -dename=${DENAME_USER} -server-host=chatterbox.xvm.mit.edu -server-pubkey=70eb7fb3e6c85c006ba76b48208ccf75f99eb6ec98dffb4292388369cb197e30

4. Create a conversation:

		chat-create -root=/home/${USER}/.chatterbox/${DENAME_USER} -subject=SubjectHere -message=hello ${DENAME_USER} DENAME_OF_FRIEND

5. Open that conversation in a terminal client:

		env EDITOR=vim cui/chat-in-tmux.sh ~/.chatterbox/${DENAME_USER}/conversations/SubjectHere\ \%between\ ${DENAME_USER}\ \%and\ DENAME_OF_FRIEND/

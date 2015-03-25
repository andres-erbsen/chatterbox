`chatterbox/client/encoding` implements a bijective, filename-safe encoding of arbitrary byte sequences. Furthermore, if we take care not to collide with percent-escaped UTF-8 codepoints and the special escape sequences in the encoding table, we can safely use "%anything" as a delimiter. For example "%between" and "and" are used to separate a conversation's name and its participants. Note that this does not limit the set of usernames we can support in any way.

Conversations in `conversations` and `outbox` are named by subject and the participants, messages are name by the sender-reported date and the sender. Messages ending in `.txt` and `.md` should be displayed as text in a GUI, other types can be just referred to. The special file `metadata.pb` in a conversation directory is not a message (TODO: get rid of it??).

If a piece of chatterbox-specific state needs to be stored on the disk, it should be placed as follows:

- If it needs to accessible to all frontends (for example, the `dename` name) should be stored in `config.pb`
- If only the daemon needs it, put it in `.daemon/config.pb`
- Secret keys should probably be moved to `.daemon/keys.pb`

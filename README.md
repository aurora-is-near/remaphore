# remaphore
 
Admin tool employing NATS to coordinate processes on distributed infrastructure.

Tasks on widely distributed machines often have to be coordinated, without opening
systems up to abuse or creating complex workflow setups.

remaphore is a "remote semaphore" that communicates via NATS message broker infrastructure
and can be used to coordinate tasks.

### Example: Upload a file to S3 and trigger mass download after upload is finished.

Server 1 (uploader):

  `$ aws s3 cp bigfile s3://bucket/ && remaphore -s -u upload_done`
  
Server 2-n (downloaders):"

  `$ remaphore -u upload_done && aws s3 cp s3://bucket/ bigfile`
  
In the above example the first server will send a remaphore message to a central
message subject when the upload has finished successfully. The other servers wait
for this message to then, and only then, start the s3 download.

**Only nodes that are connected while a message is sent can receive this message.**

## Advanced usage

remaphore exchanges signed messages over a public topic on a nats infrastructure.
One remaphore instances sends the message, the others receive it. How messages are
sent and how to react to them is a matter of configuration:

### Sending

  ```
  remaphore -s [options] [message]  
  -D string
    	Specify destination to match
  -p string
    	Use public key for sending
  -u string
    	UUID
  -v string
    	Verb to send
  ```

`-s` enables send and forget mode. The message will be sent and remaphore will
return immediately afterwards. Additional arguments control how messages are constructed.

`-D` defines a pattern that matches destination nodes. Each node contains a
`destination` in its configuration in reverse dot-segmented format, aka `tld.domain.host...`. This
destination can be matched either precisely, or by wildcard (*). Be aware that
only one wildcard per segment is allowed. The end of the pattern can be (**) to
match any number of segments after. Example:

Destination `com.crypto.host.us` can be matched by `com.crypto.host.us`, `com.crypto.*.us` or
`com.crypto.**`.

The default destination used is "**" which reaches all nodes.

`-p` can select a different public key for sending the message. A node can have
multiple identities configured that have different permissions. 

`-u` defines a (possibly unique) value in the message that receivers can match for.

`-v` defines the "verb" of a message. Each recipient has a list of peers and which
verbs those peers may use in messages. This allows authenticated access control in more complicated
scenarios.

The remainder of the commandline is considered the message payload to be sent.

### Receiving

  ```
  remaphore [options] [script]
  -D string
    	Specify destination to match
  -d	Do not match for destination
  -o	Exit after one matching message received
  -p string
    	Match for public key.
  -t duration
    	Timeout for operation
  -u string
    	UUID filter for
  -v string
    	-v <verb>[,verb...]: Verb to match 
  ```

Receiving nodes connect to the public message topic and read all messages sent.
The messages are by default filtered to be intended for the recipient node (destination).

Additional filtering can be configured:

`-D` changes the destination to match for.

`-d` disable matching for destination (dangerous).

`-u` match uuid string in message. remaphore will exit successfully on the first
matching message (implies `-o`).

`-v` match one or more verbs in a message. Only defined peers can send messages
for a specific verb. This allows for access control.

`-o` instructs remaphore to exit successfully after the first matching message
has been received. Otherwise remaphore will continue listening forever - unless
a timeout `-t` has been configured.

`-t` configures a timeout (duration format, `1h2m3s4ms`). After the timeout has
been reached, remaphore exits.

**Exit Codes**: Remaphore will return exit code 0 if it has received a message, exit code 1
if no message was received before timeout, and other exit codes on error.

Optionally remaphore can execute a script/command that is defined on the
commandline. It will be called as `cmd $verb $payload` and with these environment
variables set:

```
    REMAPHORE_SENDER    sender's public key as base58.
    REMAPHORE_VERB      the verb of the message.
    REMAPHORE_TIME      Unix timestamp when the message was sent.
    REMAPHORE_UUID      The message UUID.
    REMAPHORE_MSG       The payload of the message.
```
Only if `-d` has not been defined.
```
    REMOPHORE_DESTMATCH The matched destination.
```

By default, the output of the command will not be processed, unless the
sender requested a reply.

### Request&Response

```
  remaphore -r [options] [message]
  -t duration
  	Timeout for operation  
  ...
```

remaphore senders can request a response from the recipient(s). This behaviour
is enabled by the `-r` switch which is mutually exclusive to `-s`. In addition
to the same parameters known from send operations, the `-t` timeout parameter
is worth mentioning in this context.

remaphore maintains a list of suspected responders and will return as soon as
all responders have answered or the timeout has been reached.

Responses are written to stdout in the following format:

 `destination,exit-code,output`

One line per destination/responder.

If the reply contains newlines it will be surrounding by begin/end tags of this form:

```
--> randomchars
response
response
response
--< randomchars
```

`-->` begins a response while `--<` terminates it. The randomchars in the separator
remain the same for all responses for a request.

Responses are never interleaved.

### Additional functions

`-C` will print an example config file to stdout.

`-c string` allows specifying a config file other than the default one in `/etc/remaphore/remaphore.conf`.

`-S string` allows specifying a different NATS subject to communicate on. Needs to be
set for both sender and recipients.

## Configuration

The default configuration file is located in `/etc/remaphore/remaphore.conf`.

It looks similar to this:

```
server: nats://natsserver:4222
credentials: /path/to/credentials/file
subject: remaphore
default_identity: 3v96V3EgjiuXjmdkb5a4RjjtqfLoZCD657uyqrYZ1Xam
destination: com.crypto.us.left
allow_skew: 5s

[ Identities ]
3v9... g4xm... [ping]

[ Peers ]
com.crypto.us.right 5v22... [ping] 
```

`server` defines the NATS url to connect to. Multiple server entries can be present
for failover use.

`credentials` is the path to the nats credentials file. It is not optional.

`subject` is the default subject root to communicate on. Leave unchanged unless you understand.

`default_identity` is the public key of the default sender identity to use.

`destination` is the local destination name. It should be globally unique.

`allow_skew` defines the maximum delta between local time and time encoded in message. Messages outside the delta are ignored.

`[ Identities ]` introduces the list of locally configured identities. Each
identity consists of `publickey privatekey [verbs...]`.

Public- and private key are base58 encoded ed25519 keys. They are used to sign
messages. The `[verbs]` part is a list of comma-separated verbs that this identity can
use. When sending a message, the verb will select the identity used for sending. `[*]` means
*all verbs*.

`[ Peers ]` begins the list of known peers. Only peers configured in this list
will be able communicate to the receiving node. Each line consists of:

`destination publickey [verbs...]` 

The destination is the node's destination value, the publickey is used to authenticate
messages sent by that peer. Verbs define the verbs for which the peer may send
messages.

## Closing notes

remaphore can be started (either sending or receiving) concurrently
on the same host/same configuration. Each receiving instance will
process all messages. Be aware that only messages that are sent WHILE
the receiving node is waiting for messages will be received. Past messages
are lost in the void.

To control critical processes, make sure to define a verb so that
the senders that can trigger the receiver are limited to authorized
parties only.

In general, listeners should use the `-t` timeout option to prevent
unlimited lingering. `-t 24h` is a good safeguard.

Be aware that the remaphore configuration needs to be readable for
all users that need to send or receive messages via remaphore. This
should be used with care. It is advisable to create a group that has
read-access to the /etc/remaphore directory in exclusion of everybody else.
Multiple remaphore configuration files (and nats credentials) can be used
to limit the powers of users.

remaphore is an automation tool. Do not use it for chatting or file transfer
directly.
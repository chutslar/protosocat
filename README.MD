# protosocat

This is a TUI interface forsending and receiving
protobuf messages over websocket.

I have been using the protobuf-over-websocket architecture for another
project and been annoyed that there's no really good way to test it;
websocat is great if your message is plaintext/json but with protobuf
you have to do a lot of manual copy-paste of binary messages along with
the serde.

protosocat can serve as either the server
(for testing the client side) or the client (for testing the server side).

Usage:

```
  -b, --base-path string      path the websocket will be served on
  -c, --connect string        websocket server connection URL
  -l, --listen int            port to listen on (default -1)
      --receive-type string   full name of the type of messages to be received
```

## Common server usage

```bash
protosocat -l 3000 ./protobufs_dir/ --receive-type example.TestMessage
```

By default the server listens on `localhost:port/`, if you provide
a base path then it will listen on `localhost:port/your/base/path`.

## Common client usage

```bash
protosocat -c localhost:3000/ ./protobufs_dir/ --receive-type example.TestMessage
```

## Receive type

Both server and client usage require the name of the protobuf they will receive
to be provided (qualified by the package name). Protobuf does not have a
built in schema identifier so to correctly deserialize incoming messages
the receive type must be known.

In the future I would like to provide a way of configuring the protobuf type
via DSL like:

```
example.TestMessage: type == 4
example.OtherMessage: foo.type == 1
```

But this is complicated to implement in a robust and intuitive way,
especially for messages without a top-level type field. For now this
has not been implemented but I welcome any PRs to attempt it.

## Acknowledgements

- [Charm](https://charm.land/) - TUI library
- [Buf](https://github.com/bufbuild/protocompile) - Go protobuf compiler
- [coder/websocket](https://github.com/coder/websocket) - websocket library

# protosocat (WIP)

This is a work-in-progress project to create a TUI interface for
sending and receiving protobuf messages over websocket.

I have been using the protobuf-over-websocket architecture for another
project and been annoyed that there's no really good way to test it;
websocat is great if your message is plaintext/json but with protobuf
you have to do a lot of manual copy-paste of binary messages along with
the serde.

When this is done, you'll point to a directory with your protobufs
and you'll be able to compose your protobufs to send via a nice interface.
Messages you send and receive will be sent over the wire as binary but
saved in a TUI message history deserialized into json for easy viewing.

protosocat when done will be able to serve as either the server
(for testing the client side) or the client (for testing the server side).

## Works:

- View and create protobufs based on the definitions
- Send and receive over websocket
- Scroll through message history

## Still need to do:

- Add contextual help text

## Acknowledgements

- [Charm](https://charm.land/) - TUI library
- [Buf](https://github.com/bufbuild/protocompile) - Go protobuf compiler
- [coder/websocket](https://github.com/coder/websocket) - websocket library

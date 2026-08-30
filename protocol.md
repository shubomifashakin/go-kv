# Protocol

This is a simple protocol defining how messages are expected to be sent to the server

## Commands (Client → Server)

### SET

Store a key-value pair.

**`SET key value \r\n`**

Response: `+OK \r\n`

### GET

Retrieve a value by key.

**`GET key \r\n`**

Response: `$<length>\r\n<data>\r\n` on success, `-NOTFOUND key does not exist\r\n` if missing

### DEL

Delete a key.

**`DEL key \r\n`**

Response: `+OK \r\n` on success, `-NOTFOUND key does not exist\r\n` if missing

### EXP

Set a TTL on a key in seconds.

**`EXP key seconds \r\n`**

Response: `+OK \r\n` on success, `-NOTFOUND key does not exist\r\n` if missing

### PING

Check if the server is alive.

**`PING \r\n`**

Response: `+PONG \r\n`

## Responses (Server → Client)

| Format                    | Meaning                                 |
| ------------------------- | --------------------------------------- |
| `+OK \r\n`                | Command succeeded                       |
| `+PONG \r\n`              | Response to PING                        |
| `$<length>\r\n<data>\r\n` | Value returned from GET                 |
| `-ERRORTYPE message\r\n`  | Error — ERRORTYPE describes the failure |
| `-NOTFOUND message\r\n`   | Key does not exist                      |
| `-INVALID message\r\n`    | Malformed or unrecognised command       |

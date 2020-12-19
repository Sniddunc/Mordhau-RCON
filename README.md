# Mordhau-RCON

Mordhau-RCON is a client implementation of [Valve's Source RCON Protocol](https://developer.valvesoftware.com/wiki/Source_RCON_Protocol) written in Go. It is heavily based on my pure RCON client implementation [GoRcon](https://github.com/sniddunc/gorcon), but has some significant changes in order to support Mordhau's event broadcasting functionality.

## What is RCON?

RCON is a TCP/IP based protocol which allows commands to be sent to a game server through a remote console, which is where the term RCON comes from.

It was first used by the Source Dedicated Server to support Valve's games, but many other games implemented the idea of RCON through their own implementatins. Some notable examples are Minecraft, Rust and ARK.

## Why use RCON?

RCON is incredibly useful for game server administrators. Not only does it let them remotely execute console commands without being in game or near the server, but it can also be used to support advanced tools for automatic moderation, server monitoring and much more.

## How does Mordhau-RCON work?

Mordhau-RCON works using two TCP socket connections. One socket is for command execution, and the other is for broadcast receival. The reason two sockets are used is to make it easier to differentiate between broadcast messages and regular command execution outputs as there is currently no specific identifier for broadcasts, so we have to guess. Using two sockets provides a better chance of an accurate guess.

## How do I use it?

It's easy! First, you should create a `ClientConfig` instance and fill in the required fields. Currently, the following fields are required: `Host`, `Port`, `Password`. There also the following optional fields: `BroadcastHandler`, `SendHeartbeatCommand`, `HeartbeatCommandInterval`. For more information on config fields, check out the config section.

```go
clientConfig := &mordhaurcon.ClientConfig{
	Host:     host,
	Port:     port,
	Password: password,
	// BroadcastHandler:         broadcastHandler,
	// SendHeartbeatCommand:     true,
	// HeartbeatCommandInterval: time.Second * 10,
}

client := mordhaurcon.NewClient(clientConfig)
```

### Connecting to the RCON server

Once your client is configured to your requirements, connect the client to your RCON server using `client.Connect()`. Example:

```go
if err := client.Connect(); err != nil {
    // handle error
}
```

### Executing commands

Once theclient is connected to your RCON server, you can start sending commands using `client.ExecCommand(string)`. Example:

```go
response, err := client.ExecCommand("PlayerList")
if err != nil {
    // handle error
}

// do something with response
```

### Listening for broadcasts

To listen for broadcasts, there is an additional step. You must run the function `client.ListenForBroadcasts([]string, errors)`. This function takes in a string slice containing the broadcast channels to listen to, and `errors` is a channel where errors will be passed into incase any occur. `ListenForBroadcasts` uses goroutines which is why the error channel is needed. If you really don't care about errors, you can pass in `nil`. Example:

```go
errorChannel := make(chan error)

client.ListenForBroadcasts([]string{"all"}, errorChannel)

// Enter loop to check for errors occurred
for {
    select {
        case err := <- errors:
        // handle error
        break
    }
}
```

## Config

The following table contains a breakdown of the various config fields you can set in your `ClientConfig`.

| Field name               | Required | Description                                                                                                                                                                                                                             |
| ------------------------ | :------: | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Host                     |   Yes    | The host address of the RCON server you wish to connect to.                                                                                                                                                                             |
| Port                     |   Yes    | The port the RCON server is listening on.                                                                                                                                                                                               |
| Password                 |   Yes    | The password for the RCON server.                                                                                                                                                                                                       |
| SendHeartbeatCommand     |    No    | If this is set to true, a heartbeat command will be sent to the at the interval defined in HeartbeatCommandInterval (see below) to keep the socket alive. This is useful if your server is closing your socket connections prematurely. |
| HeartbeatCommandInterval |    No    | The interval at which to send a heartbeat command. Heartbeat commands will only be sent if `SendHeartbeatCommand` is set to true.                                                                                                       |
| BroadcastHandler         |    No    | A function matching the signature of `func(string)` which will be called each time a broadcast is received.                                                                                                                             |

## Example

For a full example, check out examples/main.go in this repository.

# Contributing

I'm always open to contributions! If you have an idea on how we can make Mordhau-RCON better, don't hesitate to reach out or open an issue or pull request!

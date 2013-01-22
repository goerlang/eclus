# Eclus #

EPMD replacement in Go

## Daemon ##

To start daemon run:

```sh
    $ eclus [-port 4369] [-nodes-limit 1000] [-unreg-ttl 10]
```

Flags:

 - `-port`: listening port for daemon, default is 4369
 - `-nodes-limit`: capacity size of nodes register, default is 1000
 - `-unreg-ttl`: time to live of inactive (down) nodes if register capacity exceed, default is 10

## CLI ##

If `eclus` cannot bind to specified port, it runs in CLI mode.

To check registered names on epmd, run `eclus` with flag `-names`:

```sh
    $ eclus -names
        asd	50249	7		active	1	Sun Dec 30 03:40:47 2012
    gangnam	40937	none	down	2	Sun Dec 30 03:44:51 2012
       oppa	36677	9		active	2	Sun Dec 30 03:44:48 2012
        qwe	60255	none	down	1	Sun Dec 30 03:44:25 2012
```

Header is: `|  Node name  |  Port of node  |  File descriptor of node connection  |  Node state  |  Creation counter  |  Recent state change date  |`


# Build Status #

[![GoCI Build Status](http://goci.me/project/image/github.com/metachord/eclus)](http://goci.me/project/github.com/metachord/eclus)

# Go-node #

Run eclus with embedded node:

```sh
    $ eclus -node -node-name 'epmd@localhost' -node-cookie 123asd [-erlang.node.trace] [-erlang.dist.trace]
```

Options `-erlang.node.trace`, `-erlang.dist.trace` will print debug info for correspond subsystems.

Then run Erlang node with the same cookie:

```sh
    $ erl -sname asd@localhost -setcookie 123asd
```

## Ping ##

Now type `net_adm:ping(epmd@localhost).` in Erlang node:

```erlang
    (asd@localhost)1> net_adm:ping(epmd@localhost).
    pong
```

You see `pong` reply from Go-node!

## Implement your own GenServer ##

See `src/eclus/esrv.go`. It is GenServer behaviour implementation which you can use like original `gen_server` process from Erlang/OTP.

To run this process first create pointer to structure which implements all methods for this behaviour:

```go
    eSrv := new(eclusSrv)
```

Then call `Spawn` method on published node:

```go
    enode.Spawn(eSrv)
```

Now you can interact with this process from Erlang-node using `gen_server:call/2`, gen_server:cast/2` or just send message to it:

```erlang
    (asd@localhost)3> gen_server:call({eclus, epmd@localhost}, message).
    {ok,eclus_reply,message}
```

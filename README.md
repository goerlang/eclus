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

If `eclus` cannot bind to specified port, it run in CLI mode.

To check registered names on epmd, run `eclus` with flag `-names`:

```sh
    $ eclus -names
        asd 50249   active  1       Sun Dec 30 03:40:47 2012
    gangnam 40937   down    2       Sun Dec 30 03:44:51 2012
       oppa 36677   active  2       Sun Dec 30 03:44:48 2012
        qwe 60255   down    1       Sun Dec 30 03:44:25 2012
```

Header is: `|  Node name  |  Port of node  |  Node state  |  Creation counter  |  Recent state change date  |`


# Build Status #

[![GoCI Build Status](http://goci.me/project/image/github.com/metachord/eclus)](http://goci.me/project/github.com/metachord/eclus)

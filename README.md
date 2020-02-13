# ipfs-connector

Using the IPFSto transfer files between computers can be slow. This makes it faster by lowering the barrier towards adding your peers to a swarm. 

First run the IPFS daemon on your computers

```
$ ipfs daemon
```

Then get `ipfs-connect` using `go`:

```
$ go get github.com/schollz/ipfs-connect
```

Then connect the two computers. On the first computer type:

```
$ ipfs-connect
your id: jkljl88-ji98-449-a0e1-c87a04802922
add another computer to your swarm by running

ipfs-connect 574ec4957f4129276db46e045e2ddf90
```

And then on the other computer add

```
$ ipfs-connect 574ec4957f4129276db46e045e2ddf90
```

And then, voila! Your computers will be swarmed together as long as the IP addresses don't change. Now you can share files via IPFS between two computers without waiting.

## How does it work?

This uses a simple rendezvous server: [schollz/duct](https://github.com/schollz/duct) which is a ephemeral MPMC pubsub. Both computers connect to the duct tape server and listen for a payload about their IPFS addresses. Once they receive the addresses they connect to them via the `ipfs swarm connect` command.

## License 

MIT
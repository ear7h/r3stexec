# esp

This is a "simple" protocol for running commands over unix sockets.

1. slave opens port
2. master sends binary encoded cmd and args to slave over unixe `.sock` addr
	* slave replies with file descriptors to pipes
3.
	* master can send signals and commands
	* master can close the connection, to which the slave kills the process
	* the process exits, and the slave sends back the exit code and closes the connection

\* step `2`, can be initiated multiple times and concurrently by accepting in a loop and calling `handleExec` as a goroutine

## notes
* closing a master's fd does not close a slave's fd. This is important when closing `stdin`
* TODO: add env var support to the initial request
```
{
	{len {cmd}}
	{len {len {arg}}}
	{len {len {key} len {val}}}
}

{} = variable length
```
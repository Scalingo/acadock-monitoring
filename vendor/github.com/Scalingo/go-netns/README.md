Network Namespace with Go
=========================

Wrapper around the `setns()` syscall to switch your application in the network
namespace of another process.

The process calling `netns.Setns()` should be root otherwise, a permission denied error
will be returned.

## Testing

The tests require Docker running on the local workstation. They have to be run as
root because of the constraints around `setns()`

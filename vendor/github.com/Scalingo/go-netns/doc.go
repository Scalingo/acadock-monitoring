package netns

/*
 * Wrapper around the setns() syscall to switch your application in the network
 * namespace of another process.
 *
 * The process calling `netns.Setns()` should be root otherwise, a permission denied error
 * will be returned.
 */

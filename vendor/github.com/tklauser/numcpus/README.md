# numcpus

[![GitHub Actions Status][1]][2]
[![Build Status][3]][4]
[![Go Report Card][5]][6]
[![GoDoc][7]][8]

Package numcpus provides information about the number of CPU.

It gets the number of CPUs (online, offline, present, possible or kernel
maximum) on a Linux, Darwin, FreeBSD, NetBSD, OpenBSD or DragonflyBSD
system.

On Linux, the information is retrieved by reading the corresponding CPU
topology files in `/sys/devices/system/cpu`.

Not all functions are supported on Darwin, FreeBSD, NetBSD, OpenBSD and
DragonflyBSD.

## Usage

```Go
package main

import (
	"fmt"
	"os"

	"github.com/tklauser/numcpus"
)

func main() {
	online, err := numcpus.GetOnline()
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetOnline: %v\n", err)
	}
	fmt.Printf("online CPUs: %v\n", online)

	possible, err := numcpus.GetPossible()
	if err != nil {
		fmt.Fprintf(os.Stderr, "GetPossible: %v\n", err)
	}
	fmt.Printf("possible CPUs: %v\n", possible)
}
```

## References

* [Linux kernel sysfs documenation for CPU attributes](https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-devices-system-cpu)
* [Linux kernel CPU topology documentation](https://www.kernel.org/doc/Documentation/cputopology.txt)

[1]: https://github.com/tklauser/numcpus/workflows/Test/badge.svg
[2]: https://github.com/tklauser/numcpus
[3]: https://travis-ci.com/tklauser/numcpus.svg?branch=master
[4]: https://travis-ci.com/tklauser/numcpus
[5]: https://goreportcard.com/badge/github.com/tklauser/numcpus
[6]: https://goreportcard.com/report/github.com/tklauser/numcpus
[7]: https://godoc.org/github.com/tklauser/numcpus?status.svg
[8]: https://godoc.org/github.com/tklauser/numcpus

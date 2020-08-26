package procfs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type CPUStat interface {
	Read(ctx context.Context) (CPUStats, error)
}

type CPUStats struct {
	CPUs map[string]SingleCPUStat
}

type SingleCPUStat struct {
	Name      string
	User      time.Duration
	Nice      time.Duration
	System    time.Duration
	IDLE      time.Duration
	IOWait    time.Duration
	IRQ       time.Duration
	SoftIRQ   time.Duration
	Steal     time.Duration
	Guest     time.Duration
	GuestNice time.Duration
}

func (c SingleCPUStat) Sum() time.Duration {
	return c.User + c.Nice + c.System + c.IDLE + c.IOWait + c.IRQ + c.SoftIRQ + c.Steal + c.Guest + c.GuestNice
}

func (c CPUStats) All() SingleCPUStat {
	return c.CPUs["cpu"]
}

type CPUStatReader struct {
	fs      FS
	sysconf Sysconf
}

func NewCPUStatReader() CPUStatReader {
	return CPUStatReader{
		fs:      NewFileSystem(),
		sysconf: NewSysconf(),
	}
}

func (c CPUStatReader) Read(ctx context.Context) (CPUStats, error) {
	result := CPUStats{
		CPUs: make(map[string]SingleCPUStat),
	}

	// Results in /proc/stats are written in a unit that can vary from system to system.
	// Usually those are 1/100 of a second.
	// We first get the unit by getting a sysconf const named SC_CLK_TCK
	clktck, err := c.sysconf.ClockTick()
	if err != nil {
		return result, errors.Wrap(err, "fail to get SC_CLK_TCK")
	}

	// Open the /proc/stat file
	file, err := c.fs.Open("/proc/stat")
	if err != nil {
		return result, errors.Wrap(err, "fail to open stat file")
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for { // For every line in the file
		line, err := reader.ReadString('\n')
		if err != nil { // If we're at the end of the file: break
			if err == io.EOF {
				break
			}
			return result, errors.Wrap(err, "fail to read a line from stat file")
		}

		// Currently this parser only cares about CPU lines There are many other things in this file but it's not supported yet
		if !strings.HasPrefix(line, "cpu") {
			continue
		}

		// Parse and store that line
		res, err := c.readOneCPULine(ctx, line, clktck)
		if err != nil {
			return result, errors.Wrap(err, "fail to parse one line of stat file")
		}
		result.CPUs[res.Name] = res
	}
	return result, nil
}

func (c CPUStatReader) readOneCPULine(ctx context.Context, line string, userHZ int64) (SingleCPUStat, error) {
	// Function called to parse a single CPU line of /proc/stat
	// Those lines look like this:
	// cpu0 13069940 9818 5731093 48473111 103287 1760557 386330 0 0 0
	// The field are: name, user, nice, system, idle, iowait, irq, soft irq, steal, guest, guest_nice
	var result SingleCPUStat

	rawBuffer := make([]uint64, 10)         // This buffer will store the raw uint64 values for every field. This is a temporary buffer that is used to parse the file. Once parsed, it will be converted to time.Duration
	timeBuffer := make([]time.Duration, 10) // This buffer will store the final values for every fields.

	// Parse the line and store it in rawBuffer
	n, err := fmt.Sscanf(line, "%s %d %d %d %d %d %d %d %d %d %d", &result.Name, &rawBuffer[0], &rawBuffer[1], &rawBuffer[2], &rawBuffer[3], &rawBuffer[4], &rawBuffer[5], &rawBuffer[6], &rawBuffer[7], &rawBuffer[8], &rawBuffer[9])
	if err != nil {
		return result, errors.Wrap(err, "fail to parse procstat line")
	}
	if n != 11 { // If we failed to parse enough value
		return result, fmt.Errorf("invalid procstat line, parsed %d field expected 11", n)
	}

	scaleFactor := time.Second / time.Duration(userHZ) // The scale factor depends on the SC_CLK_TCK sysconf.
	for i, val := range rawBuffer {                    // For every values
		timeBuffer[i] = time.Duration(val) * scaleFactor // Convert it in a time.Duration
	}
	// And store them in a struct
	result.User = timeBuffer[0]
	result.Nice = timeBuffer[1]
	result.System = timeBuffer[2]
	result.IDLE = timeBuffer[3]
	result.IOWait = timeBuffer[4]
	result.IRQ = timeBuffer[5]
	result.SoftIRQ = timeBuffer[6]
	result.Steal = timeBuffer[7]
	result.Guest = timeBuffer[8]
	result.GuestNice = timeBuffer[9]

	return result, nil
}

package procfs

import (
	"bufio"
	"context"
	"fmt"

	"github.com/Scalingo/go-utils/errors/v3"
)

var _ LoadAvg = LoadAvgReader{}

type LoadAvg interface {
	Read(ctx context.Context) (LoadAverage, error)
}

type LoadAverage struct {
	Load1          float64
	Load5          float64
	Load10         float64
	RunningProcess uint64
	TotalProcess   uint64
	LastPID        uint64
}

type LoadAvgReader struct {
	fs FS
}

func NewLoadAvgReader(ctx context.Context) LoadAvgReader {
	return LoadAvgReader{
		fs: NewFileSystem(ctx),
	}
}

func (l LoadAvgReader) Read(ctx context.Context) (LoadAverage, error) {
	res := LoadAverage{}
	// First open the file
	file, err := l.fs.Open("/proc/loadavg")
	if err != nil {
		return res, errors.Wrap(ctx, err, "open loadavg file")
	}
	defer file.Close()

	// This file consist of a single line we fetch this line and scan it
	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')
	if err != nil {
		return res, errors.Wrap(ctx, err, "read loadavg")
	}

	// This line looks like this:
	// 1.76 4.08 4.41 3/1484 2852530
	// The fields are:
	// - Load 1
	// - Load 5
	// - Load 10
	// - RunningProcess/TotalProcess
	// - Last PID
	n, err := fmt.Sscanf(line, "%f %f %f %d/%d %d", &res.Load1, &res.Load5, &res.Load10, &res.RunningProcess, &res.TotalProcess, &res.LastPID)
	if err != nil {
		return res, errors.Wrap(ctx, err, "parse loadavg line")
	}
	if n != 6 { // If this line did not have 6 fields there was an error
		return res, fmt.Errorf("invalid loadavg line, parsed %d field expected 6", n)
	}

	return res, nil
}

type LoadAvgMetricsWrapper struct {
	r LoadAvgReader
}

func FilterWrap(r LoadAvgReader) LoadAvgMetricsWrapper {
	return LoadAvgMetricsWrapper{r: r}
}

func (w LoadAvgMetricsWrapper) Read(ctx context.Context) (float64, error) {
	res, err := w.r.Read(ctx)
	if err != nil {
		return 0, err
	}
	return float64(res.RunningProcess), nil
}

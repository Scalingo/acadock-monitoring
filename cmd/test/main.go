package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Scalingo/acadock-monitoring/filters"
	"github.com/Scalingo/acadock-monitoring/procfs"
)

func main() {
	//r := procfs.NewMemInfoReader()
	//r := procfs.NewLoadAvgReader()
	//r := procfs.NewCPUStatReader()

	smoothing := filters.NewExponentialSmothing(QueueReader{
		LoadAvgReader: procfs.NewLoadAvgReader(),
	})

	go smoothing.Start(context.Background())

	for {
		time.Sleep(1 * time.Second)
		s, err := smoothing.Read(context.Background())
		if err != nil {
			fmt.Println("ERROR", err.Error())
		}
		fmt.Printf("%+v\n", s)
	}
}

type QueueReader struct {
	LoadAvgReader procfs.LoadAvgReader
}

func (q QueueReader) Read(ctx context.Context) (float64, error) {
	res, err := q.LoadAvgReader.Read(ctx)
	if err != nil {
		return 0, err
	}
	fmt.Println("Read: ", res.RunningProcess)
	return float64(res.RunningProcess), nil
}

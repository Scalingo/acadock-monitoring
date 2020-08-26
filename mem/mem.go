package mem

import (
	"os"
	"strconv"
	"sync"

	"github.com/Scalingo/acadock-monitoring/client"
	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/acadock-monitoring/docker"
	"github.com/pkg/errors"
)

const (
	LXC_MEM_USAGE_FILE      = "memory.usage_in_bytes"
	LXC_SWAP_MEM_USAGE_FILE = "memory.memsw.usage_in_bytes"
	LXC_MEM_LIMIT_FILE      = "memory.limit_in_bytes"
	LXC_SWAP_MEM_LIMIT_FILE = "memory.memsw.limit_in_bytes"
	LXC_MAX_MEM_FILE        = "memory.max_usage_in_bytes"
	LXC_MAX_SWAP_MEM_FILE   = "memory.memsw.max_usage_in_bytes"
)

type Usage struct {
	client.MemoryUsage
	SwapMemoryUsage    int64 `json:"-"`
	SwapMemoryLimit    int64 `json:"-"`
	MaxSwapMemoryUsage int64 `json:"-"`
}

type MemoryUsageGetter struct {
}

func NewMemoryUsageGetter() MemoryUsageGetter {
	return MemoryUsageGetter{}
}

func (m MemoryUsageGetter) GetMemoryUsage(id string) (Usage, error) {
	usage := Usage{}
	id, err := docker.ExpandId(id)
	if err != nil {
		return usage, errors.Wrapf(err, "fail to expand '%v'", id)
	}

	errors := make(chan error)
	wg := &sync.WaitGroup{}
	wg.Add(6)
	go m.readMemoryCgroupInt64Async(id, LXC_MAX_MEM_FILE, &usage.MaxMemoryUsage, errors, wg)
	go m.readMemoryCgroupInt64Async(id, LXC_MAX_SWAP_MEM_FILE, &usage.MaxSwapMemoryUsage, errors, wg)
	go m.readMemoryCgroupInt64Async(id, LXC_MEM_LIMIT_FILE, &usage.MemoryLimit, errors, wg)
	go m.readMemoryCgroupInt64Async(id, LXC_MEM_USAGE_FILE, &usage.MemoryUsage.MemoryUsage, errors, wg)
	go m.readMemoryCgroupInt64Async(id, LXC_SWAP_MEM_LIMIT_FILE, &usage.SwapMemoryLimit, errors, wg)
	go m.readMemoryCgroupInt64Async(id, LXC_SWAP_MEM_USAGE_FILE, &usage.SwapMemoryUsage, errors, wg)

	go func() {
		wg.Wait()
		close(errors)
	}()

	for err := range errors {
		if err != nil {
			return usage, err
		}
	}

	usage.SwapUsage = usage.SwapMemoryUsage - usage.MemoryUsage.MemoryUsage

	// As swap usage depends of memory usage, both value may result in a negative value
	// In this case, it means memory has changed between read operations and that there is no swap
	if usage.SwapUsage < 0 {
		usage.MemoryUsage.MemoryUsage = usage.SwapMemoryUsage
		usage.SwapMemoryUsage = 0
	}
	usage.SwapLimit = usage.SwapMemoryLimit - usage.MemoryLimit
	usage.MaxSwapUsage = usage.MaxSwapMemoryUsage - usage.MaxMemoryUsage
	return usage, nil
}

func (m MemoryUsageGetter) readMemoryCgroupInt64Async(id, file string, ret *int64, errors chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	val, err := m.readMemoryCgroupInt64(id, file)
	if err != nil {
		errors <- err
		return
	}
	*ret = val
}

func (m MemoryUsageGetter) readMemoryCgroupInt64(id, file string) (int64, error) {
	path := config.CgroupPath("memory", id) + "/" + file
	f, err := os.Open(path)
	if err != nil {
		return -1, errors.Wrapf(err, "fail to open '%v'", path)
	}
	defer f.Close()

	buffer := make([]byte, 16)
	n, err := f.Read(buffer)
	if err != nil {
		return -1, errors.Wrapf(err, "fail to read '%v'", path)
	}

	buffer = buffer[:n-1]
	val, err := strconv.ParseInt(string(buffer), 10, 64)
	if err != nil {
		return -1, errors.Wrapf(err, "fail to read int in '%v'", string(buffer))
	}

	return val, nil
}

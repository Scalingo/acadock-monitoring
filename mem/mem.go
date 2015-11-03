package mem

import (
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/Scalingo/acadock-monitoring/client"
	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/acadock-monitoring/docker"
)

const (
	LXC_MEM_USAGE_FILE      = "memory.usage_in_bytes"
	LXC_SWAP_MEM_USAGE_FILE = "memory.memsw.usage_in_bytes"
	LXC_MEM_LIMIT_FILE      = "memory.limit_in_bytes"
	LXC_SWAP_MEM_LIMIT_FILE = "memory.memsw.limit_in_bytes"
	LXC_MAX_MEM_FILE        = "memory.max_usage_in_bytes"
	LXC_MAX_SWAP_MEM_FILE   = "memory.memsw.max_usage_in_bytes"
)

type Usage client.MemoryUsage

func GetUsage(id string) (Usage, error) {
	usage := Usage{}
	id, err := docker.ExpandId(id)
	if err != nil {
		log.Println("Error when expanding id:", err)
		return usage, err
	}

	errors := make(chan error)
	wg := &sync.WaitGroup{}
	wg.Add(6)
	go readMemoryCgroupInt64Async(id, LXC_MAX_MEM_FILE, &usage.MaxMemoryUsage, errors, wg)
	go readMemoryCgroupInt64Async(id, LXC_MAX_SWAP_MEM_FILE, &usage.MaxSwapMemoryUsage, errors, wg)
	go readMemoryCgroupInt64Async(id, LXC_MEM_LIMIT_FILE, &usage.MemoryLimit, errors, wg)
	go readMemoryCgroupInt64Async(id, LXC_MEM_USAGE_FILE, &usage.MemoryUsage, errors, wg)
	go readMemoryCgroupInt64Async(id, LXC_SWAP_MEM_LIMIT_FILE, &usage.SwapMemoryLimit, errors, wg)
	go readMemoryCgroupInt64Async(id, LXC_SWAP_MEM_USAGE_FILE, &usage.SwapMemoryUsage, errors, wg)

	go func() {
		wg.Wait()
		close(errors)
	}()

	for err := range errors {
		if err != nil {
			return usage, err
		}
	}

	return usage, nil
}

func readMemoryCgroupInt64Async(id, file string, ret *int64, errors chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	val, err := readMemoryCgroupInt64(id, file)
	if err != nil {
		errors <- err
		return
	}
	*ret = val
}

func readMemoryCgroupInt64(id, file string) (int64, error) {
	path := config.CgroupPath("memory", id) + "/" + file
	f, err := os.Open(path)
	if err != nil {
		log.Println("Error while opening:", err)
		return -1, err
	}
	defer f.Close()

	buffer := make([]byte, 16)
	n, err := f.Read(buffer)
	if err != nil {
		log.Println("Error while reading ", path, ":", err)
		return -1, err
	}

	buffer = buffer[:n-1]
	val, err := strconv.ParseInt(string(buffer), 10, 64)
	if err != nil {
		log.Println("Error while parsing ", string(buffer), " : ", err)
		return -1, err
	}

	return val, nil
}

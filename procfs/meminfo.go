package procfs

import (
	"bufio"
	"context"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var _ Meminfo = MemInfoReader{}

type MemInfo struct {
	MemTotal     uint64 `meminfo_header:"MemTotal"`
	MemFree      uint64 `meminfo_header:"MemFree"`
	Buffers      uint64 `meminfo_header:"Buffers"`
	Cached       uint64 `meminfo_header:"Cached"`
	SwapCached   uint64 `meminfo_header:"SwapCached"`
	Active       uint64 `meminfo_header:"Active"`
	Inactive     uint64 `meminfo_header:"Inactive"`
	ActiveAnon   uint64 `meminfo_header:"Active(anon)"`
	InactiveAnon uint64 `meminfo_header:"Inactive(anon)"`
	ActiveFile   uint64 `meminfo_header:"Active(file)"`
	InactiveFile uint64 `meminfo_header:"Inactive(file)"`
	Unevictable  uint64 `meminfo_header:"Unevictable"`
	MLocked      uint64 `meminfo_header:"Mlocked"`
	SwapTotal    uint64 `meminfo_header:"SwapTotal"`
	SwapFree     uint64 `meminfo_header:"SwapFree"`
	Dirty        uint64 `meminfo_header:"Dirty"`
	Writeback    uint64 `meminfo_header:"Writeback"`
	AnonPages    uint64 `meminfo_header:"AnonPages"`
	Mapped       uint64 `meminfo_header:"Mapped"`
	Slab         uint64 `meminfo_header:"Slab"`
	SReclaimable uint64 `meminfo_header:"SReclaimable"`
	SUnreclaim   uint64 `meminfo_header:"SUnreclaim"`
	PurgeTables  uint64 `meminfo_header:"PurgeTables"`
	NFSUnstable  uint64 `meminfo_header:"NFS_Unstable"`
	Bounce       uint64 `meminfo_header:"Bounce"`
	WritebackTmp uint64 `meminfo_header:"WritebackTmp"`
	CommitLimit  uint64 `meminfo_header:"CommitLimit"`
	CommittedAS  uint64 `meminfo_header:"Committed_AS"`
	VmallocTotal uint64 `meminfo_header:"VmallocTotal"`
	VmallocUsed  uint64 `meminfo_header:"VmallocUsed"`
	VmallocChunk uint64 `meminfo_header:"VmallocChunk"`
	DirectMap4k  uint64 `meminfo_header:"DirectMap4k"`
	DirectMap2M  uint64 `meminfo_header:"DirectMap2M"`
}

type Meminfo interface {
	Read(context.Context) (MemInfo, error)
}

type MemInfoReader struct {
	fs FS
}

func NewMemInfoReader() MemInfoReader {
	return MemInfoReader{
		fs: NewFileSystem(),
	}
}

func (m MemInfoReader) Read(context.Context) (MemInfo, error) {
	res := MemInfo{}

	// Open our file
	file, err := m.fs.Open("/proc/meminfo")
	if err != nil {
		return res, errors.Wrap(err, "fail to open meminfo file")
	}
	defer file.Close()

	// This will temporary store the parsed content of the file, before being converted to a struct
	results := make(map[string]uint64)

	reader := bufio.NewReader(file)
	for {
		// Foreach line
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF { // If there is nothing to read, exit from the loop
				break
			}
			return res, errors.Wrap(err, "fail to read a line from meminfo")
		}

		// Fields will convert a single line of the meminfo file which look like this:
		// MemTotal:       15996348 kB
		// To an array which look like this:
		// ["MemTotal:", "15996348", "kB"]
		fields := strings.Fields(line)
		if len(fields) < 2 { // A line should always have a header and a value (the unit is not mandatory)
			return res, errors.Wrapf(err, "invalid meminfo line: %s", line)
		}

		fieldName := strings.TrimSuffix(fields[0], ":")    // The fieldName is the first member of the array (and remove the trailing colon)
		value, err := strconv.ParseUint(fields[1], 10, 64) // The value should come second (always positive integer values)
		if err != nil {
			return res, errors.Wrapf(err, "invalid value for %s: %s", fieldName, fields[1])
		}

		if len(fields) == 3 { // If there are a third member, it's the unit
			if fields[2] == "kB" { // The only known unit on those kind of files is kB
				value *= 1024
			} else {
				return res, errors.Wrapf(err, "invalid unit for line %s", line)
			}
		}
		results[fieldName] = value // Store the value in the temporary map
	}
	// The entire file has been parsed

	return m.FillStruct(results), nil // Transfom the map into a struct and return
}

// FilllStruct will create a fill a MemInfo struct by using it's meminfo_header tags to search the correct value in a map
func (m MemInfoReader) FillStruct(values map[string]uint64) MemInfo {
	result := MemInfo{}                      // Create the struct
	elems := reflect.ValueOf(&result).Elem() // Use reflecttivity to be able to dynamically set struct fields (Elem is used to dereference the pointer)
	types := elems.Type()                    // And to extract it's types (this will be used to list the different fields of the struct (and their tags))
	for i := 0; i < types.NumField(); i++ {  // Foreach field in the struct
		fieldType := types.Field(i)                   // Get the field type
		field := elems.FieldByName(fieldType.Name)    // Get the field element
		lookup := fieldType.Tag.Get("meminfo_header") // Get the meminfo_header tag for this field
		if lookup == "" {                             // If there is no meminfo_header tag skip this field
			continue
		}
		value, ok := values[lookup] // Search this meminfo_header tag in the struct
		if !ok {                    // If it's not found, skip it
			continue
		}
		field.SetUint(value) // Otherwise set the value in the struct
	}
	return result // And return it
}

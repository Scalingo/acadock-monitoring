package docker

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/pkg/errors"
)

func ExpandId(id string) (string, error) {
	fullId, err := expandIdFromCache(id)
	if err == nil {
		return fullId, nil
	}
	if err != IdNotInCache {
		return "", errors.Wrap(err, "fail to expand id from cache")
	}

	dir := filepath.Dir(config.CgroupPath("memory", id))
	switch config.ENV["CGROUP_SOURCE"] {
	case "systemd":
		id, err = expandSystemdId(dir, id)
	case "docker":
		id, err = expandDockerId(dir, id)
	default:
		panic("not a known CGROUP SOURCE")
	}
	if err != nil {
		return "", errors.Wrapf(err, "fail to expand id from cgroup source %v", config.ENV["CGROUP_SOURCE"])
	}

	containerIdsCache = append(containerIdsCache, id)
	return id, nil
}

func expandSystemdId(dir, id string) (string, error) {
	d, err := os.Open(dir)
	if err != nil {
		return "", errors.Wrapf(err, "fail to open '%v'", dir)
	}
	files, err := d.Readdirnames(0)
	if err != nil {
		return "", errors.Wrapf(err, "fail to read directory entries of '%v'", dir)
	}
	d.Close()
	for _, f := range files {
		if len(f) < 32 {
			continue
		}
		fullId := f[7 : len(f)-6]
		if strings.HasPrefix(fullId, id) {
			return fullId, nil
		}
	}
	return "", errors.New("id not found")
}

func expandDockerId(dir, id string) (string, error) {
	d, err := os.Open(dir)
	if err != nil {
		return "", errors.Wrapf(err, "fail to open '%v'", dir)
	}
	files, err := d.Readdirnames(0)
	if err != nil {
		return "", errors.Wrapf(err, "fail to read directory entries of '%v'", dir)
	}
	d.Close()
	for _, f := range files {
		if strings.HasPrefix(f, id) {
			return f, nil
		}
	}
	return "", errors.New("id not found")
}

package config

import (
	"os"
	"strconv"
	"time"

	"github.com/containerd/cgroups"
)

var ENV = map[string]string{
	"DOCKER_URL":                     "http://127.0.0.1:4243",
	"PORT":                           "4244",
	"REFRESH_TIME":                   "20",
	"CGROUP_SOURCE":                  "docker",
	"CGROUP_DIR":                     "/sys/fs/cgroup",
	"PROC_DIR":                       "/proc",
	"RUNNER_DIR":                     "/usr/bin",
	"DEBUG":                          "false",
	"NET_MONITORING":                 "true",
	"QUEUE_LENGTH_SAMPLING_INTERVAL": "5s",
	"QUEUE_LENGTH_POINTS_PER_SAMPLE": "5",
	"QUEUE_LENGTH_ELEMENTS_NEEDED":   "6",
	"HTTP_USERNAME":                  "",
	"HTTP_PASSWORD":                  "",
}

var (
	RefreshTime                 int
	Debug                       bool
	QueueLengthSamplingInterval time.Duration
	QueueLengthPointsPerSample  int
	QueueLengthElementsNeeded   int
	IsUsingCgroupV2             bool
)

func init() {
	for k, v := range ENV {
		if os.Getenv(k) != "" {
			ENV[k] = os.Getenv(k)
		} else {
			_ = os.Setenv(k, v)
		}
	}

	if ENV["DEBUG"] == "1" || ENV["DEBUG"] == "true" {
		Debug = true
	}

	var err error

	if cgroups.Mode() == cgroups.Unified {
		IsUsingCgroupV2 = true
	}

	RefreshTime, err = strconv.Atoi(ENV["REFRESH_TIME"])
	if err != nil {
		panic(err)
	}

	QueueLengthElementsNeeded, err = strconv.Atoi(ENV["QUEUE_LENGTH_ELEMENTS_NEEDED"])
	if err != nil {
		panic(err)
	}

	QueueLengthPointsPerSample, err = strconv.Atoi(ENV["QUEUE_LENGTH_POINTS_PER_SAMPLE"])
	if err != nil {
		panic(err)
	}

	QueueLengthSamplingInterval, err = time.ParseDuration(ENV["QUEUE_LENGTH_SAMPLING_INTERVAL"])
	if err != nil {
		panic(err)
	}

}

func CgroupPath(cgroup string, id string) string {
	if ENV["CGROUP_SOURCE"] == "docker" {
		return ENV["CGROUP_DIR"] + "/" + cgroup + "/docker/" + id
	} else if ENV["CGROUP_SOURCE"] == "systemd" {
		return ENV["CGROUP_DIR"] + "/" + cgroup + "/system.slice/docker-" + id + ".scope"
	} else {
		panic("unknown cgroup source" + ENV["CGROUP_SOURCE"])
	}
}

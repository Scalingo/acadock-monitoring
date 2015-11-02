package runner

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"

	"github.com/Scalingo/acadock-monitoring/config"
	"github.com/Scalingo/go-netns"
	"github.com/Scalingo/go-netstat"
)

var (
	netstatBinPath     string
	netstatBinName     = "acadock-monitoring-ns-netstat"
	alternativeBinPath = config.ENV["RUNNER_DIR"] + "/" + netstatBinName + "/" + netstatBinName
)

func init() {
	var err error
	netstatBinPath, err = exec.LookPath(netstatBinName)
	if err != nil {
		_, err := os.Stat(alternativeBinPath)
		if err != nil {
			panic(err)
		} else {
			netstatBinPath = alternativeBinPath
		}
	}
}

func NetStatsRunner(pid string) (netstat.NetworkStats, error) {
	ns, err := netns.SetnsFromProcDir(config.ENV["PROC_DIR"] + "/" + pid)
	if err != nil {
		return nil, err
	}
	defer ns.Close()

	stdout := new(bytes.Buffer)
	cmd := exec.Command(netstatBinPath)
	cmd.Stdout = stdout
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	err = cmd.Wait()
	if err != nil {
		return nil, err
	}
	var stats netstat.NetworkStats
	err = json.NewDecoder(stdout).Decode(&stats)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

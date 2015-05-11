package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/google/cadvisor/client"
	cadvisor "github.com/google/cadvisor/info/v1"
)

var outputFile = flag.String("output_prefix", "output", "Prefix to add to the output file.")
var host = flag.String("host", "localhost", "Host to get stats for.")
var port = flag.Int("port", 4194, "Port of the cAdvisor running on the monitored host.")
var containersToMonitor = flag.String("containers", "/,/kubelet,/docker-daemon,/kube-proxy,/system", "Comma-separated list of containers to monitor")

type StatsData struct {
	Timestamp        time.Time
	CpuCores         float64
	MemoryUsage      int64
	MemoryWorkingSet int64
}

var oldStats map[string]cadvisor.ContainerStats

func getStats(containerName string, client *client.Client) (StatsData, error) {
	request := cadvisor.ContainerInfoRequest{
		NumStats: 1,
	}
	info, err := client.ContainerInfo(containerName, &request)
	if err != nil {
		return StatsData{}, err
	}
	if len(info.Stats) == 0 {
		return StatsData{}, fmt.Errorf("received empty stats for %q", containerName)
	}
	stats := info.Stats[0]

	data := StatsData{
		Timestamp:        stats.Timestamp,
		CpuCores:         float64(stats.Cpu.Usage.Total-oldStats[containerName].Cpu.Usage.Total) / float64(stats.Timestamp.Sub(oldStats[containerName].Timestamp).Nanoseconds()),
		MemoryUsage:      int64(stats.Memory.Usage),
		MemoryWorkingSet: int64(stats.Memory.WorkingSet),
	}
	oldStats[containerName] = *stats
	return data, nil
}

func outputFirstLine(w *csv.Writer) {
	w.Write([]string{
		"Timestamp",
		"CPU Usage in Cores",
		"Memory Usage in Bytes",
		"Memory Working Set in Bytes",
	})
}

func outputLine(containerName string, stats StatsData, w *csv.Writer) {
	strOutput := []string{
		fmt.Sprintf("%v", stats.Timestamp),
		fmt.Sprintf("%.3f", stats.CpuCores),
		fmt.Sprintf("%d", stats.MemoryUsage),
		fmt.Sprintf("%d", stats.MemoryWorkingSet),
	}
	err := w.Write(strOutput)
	if err != nil {
		glog.Warningf("Failed to write stats for %q: %v", containerName, err)
	}
	w.Flush()
	glog.Infof("[%s]: %s", containerName, strings.Join(strOutput, " "))
}

func main() {
	flag.Parse()

	// Create one output file per container.
	oldStats = make(map[string]cadvisor.ContainerStats)
	now := time.Now()
	outputFiles := make(map[string]*csv.Writer)
	for _, cont := range strings.Split(*containersToMonitor, ",") {
		filename := fmt.Sprintf("%s_%s_%v.csv", *outputFile, strings.Replace(cont, "/", "", -1), now)
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			glog.Fatalf("Failed to open %q: %v", filename, err)
		}
		w := csv.NewWriter(file)
		outputFiles[cont] = w
		outputFirstLine(w)
		glog.Infof("Outputing stats for %q to %q", cont, filename)
	}

	// Create the cAdvisor client.
	client, err := client.NewClient(fmt.Sprintf("http://%s:%d/", *host, *port))
	if err != nil {
		glog.Fatalf("Failed to create cAdvisor client: %v", err)
	}

	//
	c := time.Tick(1 * time.Second)
	for range c {
		for cont, outputFile := range outputFiles {
			stats, err := getStats(cont, client)
			if err != nil {
				glog.Warningf("Failed to get stats for %q: %v", cont, err)
				continue
			}

			outputLine(cont, stats, outputFile)
		}
	}
}

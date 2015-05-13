package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/golang/glog"
)

const (
	replicationController     = "pause-controller"
	replicationControllerFile = "pause-controller.json"
)

var maxPods = flag.Int("max_pods", 50, "The max number of pods to scale to")
var podsDelta = flag.Int("pods_delta", 5, "The number of pods to change at any given step")
var waitInState = flag.Duration("wait_in_state", 10*time.Minute, "Amount of time to wait between steps")

// Run a kubectl command with the specified arguments.
func RunCommand(args ...string) {
	glog.Infof("Executing kubectl %v", args)
	out, err := exec.Command("kubectl.sh", args...).CombinedOutput()
	if err != nil {
		glog.Warningf("Failed to run %v with error: %v and output %s", args, err, string(out))
	}
}

func WriteRecord(w *csv.Writer, timestamp time.Time, event string) {
	err := w.Write([]string{
		fmt.Sprintf("%d", timestamp.Unix()),
		event,
	})
	if err != nil {
		glog.Warningf("Failed to write event %q at %v: %v", event, timestamp, err)
	}
	glog.Infof("Event %q at %v", event, timestamp)
	w.Flush()
}

func main() {
	flag.Parse()

	filename := fmt.Sprintf("output_num-pods_%v.csv", time.Now())
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		glog.Fatalf("Failed to open %q: %v", filename, err)
	}
	w := csv.NewWriter(file)
	w.Write([]string{"UNIX Timestamp", "Number of Running Pods"})

	// Start the service.
	RunCommand("create", "-f", replicationControllerFile)

	// Cleanup.
	defer func() {
		RunCommand("resize", "--replicas=0", "replicationcontrollers", replicationController)
		RunCommand("delete", "-f", replicationControllerFile)
	}()

	// Scale the replication controller up.
	for i := 0; i < (*maxPods + *podsDelta); i += *podsDelta {
		// Scale it.
		WriteRecord(w, time.Now(), fmt.Sprintf("%d", i))
		RunCommand("resize", fmt.Sprintf("--replicas=%d", i), "replicationcontrollers", replicationController)
		// TODO(vmarmol): Record events for resizing.

		// Wait.
		time.Sleep(*waitInState)
	}

	// Scale the replication controller down.
	for i := (*maxPods - *podsDelta); i >= 0; i -= *podsDelta {
		// Scale it.
		WriteRecord(w, time.Now(), fmt.Sprintf("%d", i))
		RunCommand("resize", fmt.Sprintf("--replicas=%d", i), "replicationcontrollers", replicationController)
		// TODO(vmarmol): Record events for resizing.

		// Wait.
		time.Sleep(*waitInState)
	}

	glog.Infof("Completed")
}

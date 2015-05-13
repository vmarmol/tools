package main

import (
	"flag"
	"fmt"
	"os/exec"
	"time"

	"github.com/golang/glog"
)

const (
	replicationController     = "pause-controller"
	replicationControllerFile = "pause-controller.json"
	maxPods                   = 50
	numPodsIncrement          = 5
	waitInState               = 2 * time.Minute
)

// Run a kubectl command with the specified arguments.
func RunCommand(args ...string) {
	out, err := exec.Command("kubectl.sh", args...).CombinedOutput()
	if err != nil {
		glog.Warningf("Failed to run %v with error: %v and output %s", args, err, string(out))
	}
}

func main() {
	flag.Parse()

	// Start the service.
	RunCommand("create", "-f", replicationControllerFile)

	// Cleanup.
	defer func() {
		RunCommand("resize", "--replicas=0", "replicationcontrollers", replicationController)
		RunCommand("delete", "-f", replicationControllerFile)
	}()

	// Scale the replication controller.
	for i := 0; i < (maxPods + numPodsIncrement); i += numPodsIncrement {
		// Scale it.
		glog.Infof("BEGIN --- Resize[%d]", i)
		RunCommand("resize", fmt.Sprintf("--replicas=%d", i), "replicationcontrollers", replicationController)
		// TODO(vmarmol): Should probably wait for them to be running.
		glog.Infof("END --- Resize[%d]", i)

		// Wait.
		time.Sleep(waitInState)
	}

	glog.Infof("RUN COMPLETED")
}

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/docker/libcontainer/cgroups/fs"
	"github.com/docker/libcontainer/configs"
	"github.com/golang/glog"
)

const (
	systemContainerName = "/system"
)

func main() {
	flag.Parse()

	// Create system container, move self into it.
	systemContainer := fs.Manager{
		Cgroups: &configs.Cgroup{
			Name:            systemContainerName,
			AllowAllDevices: true,
		},
	}
	systemContainer.Apply(os.Getpid())

	// Get a reference to the Root container.
	rootContainer := fs.Manager{
		Cgroups: &configs.Cgroup{
			Name: "/",
		},
	}

	// Move non-kernel PIDs to the system container.
	for {
		allPids, err := rootContainer.GetPids()
		if err != nil {
			glog.Fatalf("Failed to list PIDs for root: %v", err)
		}
		glog.Infof("Found PIDs in root: %v", allPids)

		// Remove kernel pids
		pids := make([]int, 0, len(allPids))
		for _, pid := range allPids {
			if isKernelPid(pid) {
				continue
			}

			pids = append(pids, pid)
		}

		// Check if we moved all the non-kernel PIDs.
		if len(pids) == 0 {
			break
		}

		glog.Infof("Moving non-kernel threads: %v", pids)
		for _, pid := range pids {
			err := systemContainer.Apply(pid)
			if err != nil {
				glog.Warningf("Failed to move PID %d into the system container %q: %v", pid, systemContainerName, err)
			}
		}
	}

	glog.Infof("Success!")
}

// Determines whether the specified PID is a kernel PID.
func isKernelPid(pid int) bool {
	// Kernel threads have no associated executable.
	_, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	return err != nil
}

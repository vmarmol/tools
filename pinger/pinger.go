package main

import (
	"flag"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/golang/glog"
)

type Data struct {
	LastPing time.Time
	Uptime   time.Duration
	Errors   int

	pid int
}

var kubeletData Data
var dockerData Data

func kubeletPinger() {
	transport := http.Transport{
		Dial: dialTimeout,
	}

	client := http.Client{
		Transport: &transport,
	}
	start := time.Now()

	c := time.Tick(500 * time.Millisecond)
	for _ = range c {
		// Check Kubelet.
		resp, err := client.Get("http://localhost:10255/healthz")
		if err != nil || resp.StatusCode != http.StatusOK {
			kubeletData.Errors++
			glog.Infof("Kubelet error: %v (%v)", err, resp.StatusCode)
			continue
		}
		kubeletData.LastPing = time.Now()

		// Get uptime.
		p := getPid("kubelet")
		if p != 0 {
			if kubeletData.pid != p {
				start = time.Now()
				kubeletData.pid = p
			}

			kubeletData.Uptime = time.Since(start)
		}
	}
}

func dockerPinger() {
	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)
	if err != nil {
		glog.Fatal(err)
	}

	start := time.Now()

	c := time.Tick(500 * time.Millisecond)
	for _ = range c {
		// Check Docker.
		_, err := client.Version()
		if err != nil {
			dockerData.Errors++
			glog.Infof("Docker error: %v", err)
			continue
		}
		dockerData.LastPing = time.Now()

		// Get uptime.
		p := getPid("docker")
		if p != 0 {
			if dockerData.pid != p {
				start = time.Now()
				dockerData.pid = p
			}

			dockerData.Uptime = time.Since(start)
		}
	}
}

func getPid(bin string) int {
	out, err := exec.Command("pidof", bin).CombinedOutput()
	if err != nil {
		glog.Infof("Failed to run pidof: %v (Output: %s)", err, string(out))
		return 0
	}
	i, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		glog.Infof("Failed to find pid from %q", string(out))
		return 0
	}
	return i
}

var timeout = time.Duration(1 * time.Second)

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

func main() {
	flag.Parse()

	go kubeletPinger()
	go dockerPinger()

	c := time.Tick(1 * time.Second)
	kubeletLastUptime := time.Duration(0)
	dockerLastUptime := time.Duration(0)
	for _ = range c {
		kubeletLastPing := time.Since(kubeletData.LastPing)
		if kubeletLastPing > 600*time.Millisecond {
			glog.Infof("Kubelet high ping: %v", kubeletLastPing)
		}
		if kubeletData.Uptime < kubeletLastUptime {
			glog.Infof("Kubelet restarted after %v", kubeletLastUptime)
		}
		kubeletLastUptime = kubeletData.Uptime

		dockerLastPing := time.Since(dockerData.LastPing)
		if dockerLastPing > 600*time.Millisecond {
			glog.Infof("Docker high ping: %v", dockerLastPing)
		}
		if dockerData.Uptime < dockerLastUptime {
			glog.Infof("Docker restarted after %v", dockerLastUptime)
		}
		dockerLastUptime = dockerData.Uptime
	}
}

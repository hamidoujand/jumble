package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"time"
)

type Container struct {
	Name     string
	HostPort string
}

type NetworkSettings struct {
	Ports map[string][]struct {
		HostIP   string `json:"hostIp"`
		HostPort string `json:"hostPort"`
	} `json:"ports"`
}

type ContainerInfo struct {
	NetworkSettings NetworkSettings `json:"NetworkSettings"`
}

func StartContainer(image string, name string, port string, dockerArgs []string, containerArgs []string) (Container, error) {
	//2 retries
	for i := range 2 {
		c, err := startContainer(image, name, port, dockerArgs, containerArgs)
		if err != nil {
			//sleep
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
			continue
		}
		return c, nil
	}

	return startContainer(image, name, port, dockerArgs, containerArgs)
}

func StopContainer(container string) error {
	//graceful stop
	if err := exec.Command("docker", "stop", container).Run(); err != nil {
		return fmt.Errorf("failed to stop container %s: %s", container, err)
	}

	//remove volumes
	if err := exec.Command("docker", "rm", container, "-v").Run(); err != nil {
		return fmt.Errorf("failed to remove volume %s: %s", container, err)
	}

	return nil
}

func DumpContainerLogs(container string) []byte {
	out, err := exec.Command("docker", "logs", container).CombinedOutput()
	if err != nil {
		return nil
	}
	return out
}

func startContainer(image string, name string, port string, dockerArgs []string, containerArgs []string) (Container, error) {
	//check to see if the container with this id/name is not running at the moment
	if c, err := exists(name, port); err == nil {
		return c, nil
	}

	//if there is a container stopped with the same name we want to delete it first
	_ = exec.Command("docker", "rm", name, "-v").Run()

	args := []string{"run", "-P", "-d", "--name", name}
	args = append(args, dockerArgs...)
	args = append(args, image)
	args = append(args, containerArgs...)

	command := exec.Command("docker", args...)
	var out bytes.Buffer
	command.Stdout = &out
	if err := command.Run(); err != nil {
		return Container{}, fmt.Errorf("running docker command failed: %w", err)
	}

	id := out.String()[:12]
	hostIP, hostPort, err := extractIPPort(id, port)
	if err != nil {
		_ = StopContainer(id)
		return Container{}, fmt.Errorf("extract IP port failed: %w", err)
	}

	return Container{Name: name, HostPort: net.JoinHostPort(hostIP, hostPort)}, nil
}

func extractIPPort(containerId string, port string) (hostIP string, hostPort string, err error) {
	command := exec.Command("docker", "inspect", containerId)

	var output bytes.Buffer
	command.Stdout = &output

	if err := command.Run(); err != nil {
		return "", "", fmt.Errorf("running docker inspect on container %s: %w", containerId, err)
	}

	var containerInfos []ContainerInfo
	if decodeErr := json.NewDecoder(&output).Decode(&containerInfos); decodeErr != nil {
		return "", "", fmt.Errorf("decoding containerInfos: %w", err)
	}

	//since we are inspecting using containerID, it will return 1 container
	key := port + "/tcp"
	ports := containerInfos[0].NetworkSettings.Ports[key]

	for _, host := range ports {
		if host.HostIP != "::" { //skip IPv6
			if host.HostIP == "" {
				return "localhost", hostPort, nil
			}
			return host.HostIP, host.HostPort, nil
		}
	}

	return "", "", fmt.Errorf("host:port not found for container %s", containerId)
}

func exists(name string, port string) (Container, error) {
	hostIP, hostPort, err := extractIPPort(name, port)
	if err == nil {
		return Container{
			Name:     name,
			HostPort: net.JoinHostPort(hostIP, hostPort),
		}, nil
	}

	return Container{}, fmt.Errorf("container with name/id %s not found", name)
}

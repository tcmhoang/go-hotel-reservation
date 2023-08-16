// Package docker provides support for spinning a docker img in the shell
package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

type Container struct {
	ID   string
	Host string
}

func InitContainer(img string, port string, dockargs []string, appargs []string) (*Container, error) {
	args := []string{"run", "-P", "-d"}
	args = append(args, dockargs...)
	args = append(args, appargs...)

	cmd := exec.Command("docker", args...)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("could not start container %s: %w", image, err)
	}

	id := out.String()[:12]

	eip, eport, err := auxGetIPPort(id, port)
	if err != nil {
		StopContainer(id)
		return nil, err
	}

	c := Container{
		ID:   id,
		Host: net.JoinHostPort(eip, eport),
	}

	return &c, nil
}

func auxGetIPPort(id string, iport string) (extip string, extport string, err error) {
	templstr := fmt.Sprintf("[{{range $k,$v := (index .NetworkSettings.Ports \"%s/tcp\")}}{{json $v}}{{end}}]", iport)

	cmd := exec.Command("docker", "inspect", "-f", templstr, id)
	var o bytes.Buffer
	cmd.Stdout = &o

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("could not inspect container %s: %w", id, err)
	}

	rawjson := strings.ReplaceAll(o.String(), "}{", "},{")

	var docs []struct {
		HostIP   string
		HostPort string
	}

	if err := json.Unmarshal([]byte(rawjson), &docs); err != nil {
		return "", "", fmt.Errorf("could not decode json: %w", err)
	}

	for _, doc := range docs {
		if doc.HostIP != "::" {
			return doc.HostIP, doc.HostPort, nil
		}
	}

	return "", "", fmt.Errorf("fail to locate ip/port")

}

func StopContainer(id string) error {
	if err := exec.Command("docker", "stop", id).Run(); err != nil {
		return fmt.Errorf("could not stop container: %w", err)
	}

	if err := exec.Command("docker", "rm", id, "-v").Run(); err != nil {
		return fmt.Errorf("could not remove container: %w", err)
	}

	return nil
}

func DumpContainerLogs(id string) []byte {
	out, err := exec.Command("docker", "logs", id).CombinedOutput()
	if err != nil {
		return nil
	}
	return out
}

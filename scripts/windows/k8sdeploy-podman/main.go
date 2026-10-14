package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const defaultImage = "quay.io/migqe/k8s-apps-deployer:multiarch"

func main() {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "resolve user home: %v\n", err)
			os.Exit(1)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	image := os.Getenv("K8SDEPLOY_IMAGE")
	if image == "" {
		image = defaultImage
	}

	args := []string{
		"run", "--rm", "--pull=missing",
		"--mount", fmt.Sprintf("type=bind,source=%s,target=/kube/config,readonly", kubeconfig),
		"--env", "KUBECONFIG=/kube/config",
		image,
	}
	args = append(args, os.Args[1:]...)

	cmd := exec.Command("podman", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "run podman: %v\n", err)
		os.Exit(1)
	}
}

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultImage = "quay.io/migqe/k8s-apps-deployer:multiarch"
const containerKubeconfig = "/kube/config"

func main() {
	image := os.Getenv("K8SDEPLOY_IMAGE")
	if image == "" {
		image = defaultImage
	}

	kubeconfig, explicitKubeconfig := kubeconfigPath(os.Args[1:])
	token := os.Getenv("K8S_TOKEN")
	server := os.Getenv("K8S_SERVER")
	if token == "" {
		token = optionValue(os.Args[1:], "--token")
	}
	if server == "" {
		server = optionValue(os.Args[1:], "--server")
	}

	useTokenAuth := token != "" && server != ""
	mountKubeconfig := kubeconfig != "" && (!useTokenAuth || explicitKubeconfig)
	if mountKubeconfig {
		if _, err := os.Stat(kubeconfig); err != nil {
			fmt.Fprintf(os.Stderr, "stat kubeconfig %q: %v\n", kubeconfig, err)
			os.Exit(1)
		}
	}
	if !useTokenAuth && !mountKubeconfig {
		fmt.Fprintln(os.Stderr, "kubeconfig is required unless both K8S_TOKEN and K8S_SERVER or --token and --server are provided")
		os.Exit(1)
	}

	containerArgs := replaceKubeconfigArg(os.Args[1:])
	var flattenedKubeconfig string
	if mountKubeconfig {
		var err error
		flattenedKubeconfig, err = flattenKubeconfig(kubeconfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "flatten kubeconfig %q: %v\n", kubeconfig, err)
			os.Exit(1)
		}
		defer os.Remove(flattenedKubeconfig)
	}

	args := []string{
		"run", "--rm", "--pull=missing",
	}
	if mountKubeconfig {
		args = append(args,
			"--mount", fmt.Sprintf("type=bind,source=%s,target=%s,readonly", flattenedKubeconfig, containerKubeconfig),
			"--env", "KUBECONFIG="+containerKubeconfig,
		)
	}
	if value := os.Getenv("K8S_TOKEN"); value != "" {
		args = append(args, "--env", "K8S_TOKEN")
	}
	if value := os.Getenv("K8S_SERVER"); value != "" {
		args = append(args, "--env", "K8S_SERVER")
	}
	args = append(args, image)
	args = append(args, containerArgs...)

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

func kubeconfigPath(args []string) (string, bool) {
	if value, ok := optionValueWithPresence(args, "--kubeconfig"); ok {
		return value, true
	}

	value := os.Getenv("KUBECONFIG")
	if value != "" {
		return value, false
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(home, ".kube", "config"), false
}

func optionValue(args []string, option string) string {
	value, _ := optionValueWithPresence(args, option)
	return value
}

func optionValueWithPresence(args []string, option string) (string, bool) {
	prefix := option + "="
	for index, arg := range args {
		if arg == option && index+1 < len(args) {
			return args[index+1], true
		}
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix), true
		}
	}
	return "", false
}

func replaceKubeconfigArg(args []string) []string {
	result := append([]string(nil), args...)
	for index, arg := range result {
		if arg == "--kubeconfig" && index+1 < len(result) {
			result[index+1] = containerKubeconfig
			continue
		}
		if strings.HasPrefix(arg, "--kubeconfig=") {
			result[index] = "--kubeconfig=" + containerKubeconfig
		}
	}
	return result
}

func flattenKubeconfig(kubeconfig string) (string, error) {
	command := exec.Command("kubectl", "config", "view", "--raw", "--flatten", "--kubeconfig", kubeconfig)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("kubectl config view failed: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("kubectl config view failed: %w", err)
	}

	temporaryFile, err := os.CreateTemp("", "k8sdeploy-kubeconfig-*.yaml")
	if err != nil {
		return "", fmt.Errorf("create temporary kubeconfig: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	if _, err := temporaryFile.Write(output); err != nil {
		temporaryFile.Close()
		os.Remove(temporaryPath)
		return "", fmt.Errorf("write temporary kubeconfig: %w", err)
	}
	if err := temporaryFile.Close(); err != nil {
		os.Remove(temporaryPath)
		return "", fmt.Errorf("close temporary kubeconfig: %w", err)
	}
	return temporaryPath, nil
}

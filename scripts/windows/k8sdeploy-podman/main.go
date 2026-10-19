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

	kubeconfig, _ := kubeconfigPath(os.Args[1:])
	token := os.Getenv("K8S_TOKEN")
	server := os.Getenv("K8S_SERVER")
	if token == "" {
		token = optionValue(os.Args[1:], "--token")
	}
	if server == "" {
		server = optionValue(os.Args[1:], "--server")
	}

	useTokenAuth := token != "" && server != ""
	mountKubeconfig := kubeconfig != "" || useTokenAuth
	if !mountKubeconfig {
		fmt.Fprintln(os.Stderr, "kubeconfig is required unless both K8S_TOKEN and K8S_SERVER or --token and --server are provided")
		os.Exit(1)
	}

	containerArgs := replaceKubeconfigArg(os.Args[1:])
	var mountedKubeconfig string
	if useTokenAuth {
		var err error
		mountedKubeconfig, err = createTokenKubeconfig(server, token, hasOption(os.Args[1:], "--insecure-skip-tls-verify"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "create token kubeconfig: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(mountedKubeconfig)
	} else {
		var err error
		mountedKubeconfig, err = flattenKubeconfig(kubeconfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "flatten kubeconfig %q: %v\n", kubeconfig, err)
			os.Exit(1)
		}
		defer os.Remove(mountedKubeconfig)
	}

	args := []string{
		"run", "--rm", "--pull=missing",
	}
	if mountKubeconfig {
		args = append(args,
			"--mount", fmt.Sprintf("type=bind,source=%s,target=%s,readonly", mountedKubeconfig, containerKubeconfig),
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

func hasOption(args []string, option string) bool {
	for _, arg := range args {
		if arg == option {
			return true
		}
	}
	return false
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
	paths := filepath.SplitList(kubeconfig)
	if len(paths) == 0 {
		return "", fmt.Errorf("kubeconfig path is empty")
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("stat kubeconfig %q: %w", path, err)
		}
	}

	command := exec.Command("kubectl", "config", "view", "--raw", "--flatten")
	command.Env = append(os.Environ(), "KUBECONFIG="+strings.Join(paths, string(os.PathListSeparator)))
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

func createTokenKubeconfig(server, token string, insecureSkipTLSVerify bool) (string, error) {
	temporaryFile, err := os.CreateTemp("", "k8sdeploy-token-kubeconfig-*.yaml")
	if err != nil {
		return "", fmt.Errorf("create temporary kubeconfig: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	if err := temporaryFile.Close(); err != nil {
		os.Remove(temporaryPath)
		return "", fmt.Errorf("close temporary kubeconfig: %w", err)
	}
	os.Remove(temporaryPath)

	clusterArgs := []string{
		"config", "set-cluster", "k8sdeploy",
		"--server=" + server,
		"--kubeconfig", temporaryPath,
	}
	if insecureSkipTLSVerify {
		clusterArgs = append(clusterArgs, "--insecure-skip-tls-verify=true")
	}
	commands := [][]string{
		clusterArgs,
		{"config", "set-credentials", "k8sdeploy", "--token=" + token, "--kubeconfig", temporaryPath},
		{"config", "set-context", "k8sdeploy", "--cluster=k8sdeploy", "--user=k8sdeploy", "--kubeconfig", temporaryPath},
		{"config", "use-context", "k8sdeploy", "--kubeconfig", temporaryPath},
	}
	for _, args := range commands {
		if output, err := runKubectl(args...); err != nil {
			os.Remove(temporaryPath)
			return "", fmt.Errorf("kubectl %s: %w: %s", args[1], err, strings.TrimSpace(output))
		}
	}
	return temporaryPath, nil
}

func runKubectl(args ...string) (string, error) {
	command := exec.Command("kubectl", args...)
	output, err := command.CombinedOutput()
	return string(output), err
}

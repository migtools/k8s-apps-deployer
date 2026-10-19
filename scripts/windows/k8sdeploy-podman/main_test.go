package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOptionValue(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		option string
		want   string
	}{
		{name: "separate value", args: []string{"--token", "secret"}, option: "--token", want: "secret"},
		{name: "equals value", args: []string{"--server=https://api.example.com"}, option: "--server", want: "https://api.example.com"},
		{name: "missing", args: []string{"list"}, option: "--token", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := optionValue(test.args, test.option); got != test.want {
				t.Fatalf("optionValue() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestReplaceKubeconfigArg(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "separate value",
			args: []string{"--kubeconfig", `C:\Users\test\config`, "list"},
			want: []string{"--kubeconfig", "/kube/config", "list"},
		},
		{
			name: "equals value",
			args: []string{"--kubeconfig=C:\\Users\\test\\config", "list"},
			want: []string{"--kubeconfig=/kube/config", "list"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := replaceKubeconfigArg(test.args); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("replaceKubeconfigArg() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestKubeconfigPathList(t *testing.T) {
	first := filepath.Join(t.TempDir(), "source-config")
	second := filepath.Join(t.TempDir(), "target-config")
	value := strings.Join([]string{first, second}, string(filepath.ListSeparator))

	paths := filepath.SplitList(value)
	if !reflect.DeepEqual(paths, []string{first, second}) {
		t.Fatalf("filepath.SplitList() = %#v, want %#v", paths, []string{first, second})
	}
}

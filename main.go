package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func PrintUsageAndExit() {
	fmt.Println("Usage: with-config command")
	fmt.Println("  command: command to execute")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  WITH_CONFIG: Kubernetes ConfigMap name")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  $ export WITH_CONFIG=my-configmap")
	fmt.Println("  $ with-config my-command")
	os.Exit(1)
}

func getConfigMap(namespace, configMapName string) (map[string]string, error) {
	// var kubeconfig string
	// if home := homedir.HomeDir(); home != "" {
	// 	kubeconfig = filepath.Join(home, ".kube", "config")
	// } else {
	// 	return nil, fmt.Errorf("could not find kubeconfig file")
	// }

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// if you want to change the loading rules (which files in which order), you can do so here

	configOverrides := &clientcmd.ConfigOverrides{}
	// if you want to change override values or bind them to flags, there are methods to help you

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()

	// config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	// if err != nil {
	// 	return nil, err
	// }

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	if namespace == "" {
		namespace, _, err = kubeConfig.Namespace()
		if err != nil {
			namespace = "default"
		}

		// fmt.Printf("Namespace not provided. Using the default namespace: %s\n", namespace)
	}

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return configMap.Data, nil
}

func main() {
	// check if WITH_CONFIG is set
	configmap_name := os.Getenv("WITH_CONFIG")

	if configmap_name == "" {
		fmt.Fprintln(os.Stderr, "[with-config] Error: WITH_CONFIG is not set")
		PrintUsageAndExit()
	}

	// check if command is provided
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "[with-config] Error: command is not provided")
		PrintUsageAndExit()
	}

	// Fetch the ConfigMap from Kubernetes
	namespace := os.Getenv("KUBE_NAMESPACE")

	configmap_data, err := getConfigMap(namespace, configmap_name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[with-config] Error fetching ConfigMap: %+v\n", err)
		os.Exit(1)
	}

	// execute command with secret
	cmd := os.Args[1]
	args := os.Args[2:]

	command := exec.Command(cmd, args...)

	// set environment variables
	command.Env = os.Environ()
	for key, value := range configmap_data {
		command.Env = append(command.Env, key+"="+value)
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[with-config] Error: unable to get stdout pipe: %+v\n", err)
		os.Exit(1)
	}

	stderr, err := command.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[with-config] Error: unable to get stderr pipe: %+v\n", err)
		os.Exit(1)
	}

	if err := command.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "[with-config] Error: unable to start command: %+v\n", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		scanner.Split(bufio.ScanRunes)
		for scanner.Scan() {
			fmt.Print(scanner.Text())
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		scanner.Split(bufio.ScanRunes)

		for scanner.Scan() {
			fmt.Print(scanner.Text())
		}
	}()

	wg.Wait()

	if err := command.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "[with-config] Error: command execution failed: %+v\n", err)
		os.Exit(1)
	}
}

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

const (
	PLAIN_PRINT_FIRST_N_CHARS = 4
)

func PrintUsageAndExit() {
	fmt.Println("Usage: with-configmap command")
	fmt.Println("  command: command to execute")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  WITH_CONFIGMAP: Kubernetes ConfigMap name")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  $ export WITH_CONFIGMAP=my-configmap")
	fmt.Println("  $ with-configmap my-command")
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

		fmt.Printf("Namespace not provided. Using the default namespace: %s\n", namespace)
	}

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return configMap.Data, nil
}

func main() {
	// check if WITH_CONFIGMAP is set
	configmap_name := os.Getenv("WITH_CONFIGMAP")

	if configmap_name == "" {
		fmt.Println("Error: WITH_CONFIGMAP is not set")
		PrintUsageAndExit()
	}

	// check if command is provided
	if len(os.Args) < 2 {
		fmt.Println("Error: command is not provided")
		PrintUsageAndExit()
	}

	// Fetch the ConfigMap from Kubernetes
	namespace := os.Getenv("KUBE_NAMESPACE")

	configmap_data, err := getConfigMap(namespace, configmap_name)
	if err != nil {
		fmt.Println("Error fetching ConfigMap:", err)
		os.Exit(1)
	}

	// execute command with secret
	cmd := os.Args[1]
	args := os.Args[2:]

	command := exec.Command(cmd, args...)

	// set environment variables
	for key, value := range configmap_data {
		command.Env = append(command.Env, key+"="+value)
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		fmt.Println("Error: unable to get stdout pipe")
		fmt.Println(err)
		os.Exit(1)
	}

	stderr, err := command.StderrPipe()
	if err != nil {
		fmt.Println("Error: unable to get stderr pipe")
		fmt.Println(err)
		os.Exit(1)
	}

	if err := command.Start(); err != nil {
		fmt.Println("Error: unable to start command")
		fmt.Println(err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		scanner.Split(bufio.ScanBytes)
		for scanner.Scan() {
			fmt.Print(scanner.Text())
		}
		fmt.Println("EoF: stdout")
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		scanner.Split(bufio.ScanBytes)

		for scanner.Scan() {
			for scanner.Scan() {
				fmt.Print(scanner.Text())
			}
			fmt.Println("EoF: stdout")
			// fmt.Println("EoF: stderr")
		}
	}()

	wg.Wait()

	if err := command.Wait(); err != nil {
		fmt.Println("Error: command execution failed")
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Command executed successfully")
}

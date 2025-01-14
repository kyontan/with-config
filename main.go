package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

const (
	PLAIN_PRINT_FIRST_N_CHARS = 4
)

func PrintUsageAndExit() {
	fmt.Println("Usage: with-secrets command")
	fmt.Println("  command: command to execute")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  WITH_SECRETS_ID: AWS Secrets Manager ID")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  $ export WITH_SECRETS_ID=my-secret-id")
	fmt.Println("  $ with-secrets my-command")
	os.Exit(1)
}

func GetSecretFromAWS(secret_id string) string {
	// get secret from AWS Secrets Manager
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secret_id),
	}

	sess := session.Must(session.NewSession())
	svc := secretsmanager.New(sess)

	result, err := svc.GetSecretValue(input)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return *result.SecretString
}

func main() {
	// check if WITH_SECRETS_ID is set
	secret_id := os.Getenv("WITH_SECRETS_ID")

	if secret_id == "" {
		fmt.Println("Error: WITH_SECRETS_ID is not set")
		PrintUsageAndExit()
	}

	// check if command is provided
	if len(os.Args) < 2 {
		fmt.Println("Error: command is not provided")
		PrintUsageAndExit()
	}

	// get secret from AWS Secrets Manager
	secret := GetSecretFromAWS(secret_id)

	// assumption: secret is JSON and has key-value pairs
	// parse secret JSON
	secret_json := []byte(secret)
	secret_map := make(map[string]string)
	err := json.Unmarshal(secret_json, &secret_map)
	if err != nil {
		fmt.Println("Error: secret is not JSON or not key-value pairs")
		fmt.Println(err)
		os.Exit(1)
	}

	// Build the trie-tree from secret_map values
	trie := BuildTrieFromSecretsMap(secret_map)
	// trie := NewTrieTree() // no mask

	// execute command with secret
	cmd := os.Args[1]
	args := os.Args[2:]

	command := exec.Command(cmd, args...)

	// set environment variables
	for key, value := range secret_map {
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
		trie_state := NewTrieNodeState()
		var remaining string
		for scanner.Scan() {
			var masked string
			line := scanner.Text()
			remaining += line
			masked, remaining, trie_state = trie.Mask(remaining, trie_state)
			fmt.Print(masked)
			if len(masked) > 0 {
				fmt.Print("@")
			}
		}
		// masked := trie.PrintRemaining(remaining, trie_state)
		// fmt.Print(masked)
		fmt.Println("EoF: stdout")
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		scanner.Split(bufio.ScanBytes)

		trie_state := NewTrieNodeState()
		var remaining string
		for scanner.Scan() {
			var masked string
			line := scanner.Text()
			remaining += line
			masked, remaining, trie_state = trie.Mask(remaining, trie_state)
			fmt.Print(masked)
		}
		masked := trie.PrintRemaining(remaining, trie_state)
		fmt.Print(masked)
		fmt.Println("EoF: stderr")
	}()

	wg.Wait()

	if err := command.Wait(); err != nil {
		fmt.Println("Error: command execution failed")
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Command executed successfully")
}

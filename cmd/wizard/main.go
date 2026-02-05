package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/grs/mcp-deployment/pkg/deployer"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Create Kubernetes client
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	if envKubeconfig := os.Getenv("KUBECONFIG"); envKubeconfig != "" {
		kubeconfig = envKubeconfig
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Failed to build config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	mcpDeployer := deployer.NewSimpleDeployer(clientset)

	// Main menu
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\n=== MCP Server Deployment Wizard ===")
		fmt.Println("1. List MCP servers")
		fmt.Println("2. Deploy new MCP server")
		fmt.Println("3. Delete MCP server")
		fmt.Println("4. Exit")
		fmt.Print("\nSelect an option: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			listServers(mcpDeployer, reader)
		case "2":
			deployServer(mcpDeployer, reader)
		case "3":
			deleteServer(mcpDeployer, reader)
		case "4":
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Invalid option. Please try again.")
		}
	}
}

func listServers(mcpDeployer *deployer.SimpleDeployer, reader *bufio.Reader) {
	fmt.Print("\nEnter namespace (default): ")
	namespace, _ := reader.ReadString('\n')
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		namespace = "default"
	}

	servers, err := mcpDeployer.ListMCPServers(context.Background(), namespace)
	if err != nil {
		fmt.Printf("Error listing servers: %v\n", err)
		return
	}

	if len(servers) == 0 {
		fmt.Printf("\nNo MCP servers found in namespace '%s'\n", namespace)
		return
	}

	fmt.Printf("\n=== MCP Servers in namespace '%s' ===\n\n", namespace)
	for i, server := range servers {
		fmt.Printf("Server %d:\n", i+1)
		fmt.Printf("  Name:      %s\n", server.Name)
		fmt.Printf("  Namespace: %s\n", server.Namespace)
		fmt.Printf("  Image:     %s\n", server.Image)
		fmt.Printf("  Available: %t\n", server.Available)
		fmt.Printf("  Endpoint:  %s\n", server.Endpoint)

		if len(server.Labels) > 0 {
			fmt.Println("  Labels:")
			for k, v := range server.Labels {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}

		if len(server.Annotations) > 0 {
			fmt.Println("  Annotations:")
			for k, v := range server.Annotations {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}

		if len(server.Conditions) > 0 {
			fmt.Println("  Conditions:")
			for _, condition := range server.Conditions {
				fmt.Printf("    - %s\n", condition)
			}
		}
		fmt.Println()
	}
}

func deployServer(mcpDeployer *deployer.SimpleDeployer, reader *bufio.Reader) {
	fmt.Println("\n=== Deploy New MCP Server ===\n")

	spec := &deployer.MCPServerSpec{
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
	}

	// Name
	fmt.Print("Enter MCP server name: ")
	spec.Name, _ = reader.ReadString('\n')
	spec.Name = strings.TrimSpace(spec.Name)
	if spec.Name == "" {
		fmt.Println("Error: Name is required")
		return
	}

	// Namespace
	fmt.Print("Enter namespace (default): ")
	spec.Namespace, _ = reader.ReadString('\n')
	spec.Namespace = strings.TrimSpace(spec.Namespace)
	if spec.Namespace == "" {
		spec.Namespace = "default"
	}

	// Image
	fmt.Print("Enter container image: ")
	spec.Image, _ = reader.ReadString('\n')
	spec.Image = strings.TrimSpace(spec.Image)
	if spec.Image == "" {
		fmt.Println("Error: Image is required")
		return
	}

	// Port
	fmt.Print("Enter port number (8080): ")
	portStr, _ := reader.ReadString('\n')
	portStr = strings.TrimSpace(portStr)
	if portStr == "" {
		spec.Port = 8080
	} else {
		port, err := strconv.ParseInt(portStr, 10, 32)
		if err != nil {
			fmt.Printf("Error: Invalid port number: %v\n", err)
			return
		}
		spec.Port = int32(port)
	}

	// Environment Variables
	spec.EnvVars = promptForEnvVars(reader)

	// Args
	spec.Args = promptForArgs(reader)

	// Secret Mounts
	spec.SecretMounts = promptForSecretMounts(reader)

	// Service Account
	fmt.Print("Enter service account (leave empty for default): ")
	spec.ServiceAccount, _ = reader.ReadString('\n')
	spec.ServiceAccount = strings.TrimSpace(spec.ServiceAccount)

	// Labels
	spec.Labels = promptForKeyValuePairs(reader, "label")

	// Annotations
	spec.Annotations = promptForKeyValuePairs(reader, "annotation")

	// Resource limits and requests
	spec.Resources = promptForResources(reader)

	// Confirm deployment
	fmt.Println("\n=== Deployment Summary ===")
	fmt.Printf("Name:           %s\n", spec.Name)
	fmt.Printf("Namespace:      %s\n", spec.Namespace)
	fmt.Printf("Image:          %s\n", spec.Image)
	fmt.Printf("Port:           %d\n", spec.Port)
	fmt.Printf("Service Account: %s\n", spec.ServiceAccount)
	fmt.Printf("Env Vars:       %d\n", len(spec.EnvVars))
	fmt.Printf("Args:           %d\n", len(spec.Args))
	fmt.Printf("Secret Mounts:  %d\n", len(spec.SecretMounts))
	fmt.Printf("Labels:         %d\n", len(spec.Labels))
	fmt.Printf("Annotations:    %d\n", len(spec.Annotations))
	if spec.Resources != nil {
		fmt.Println("Resources:      configured")
		if len(spec.Resources.Requests) > 0 {
			fmt.Printf("  Requests:     %v\n", spec.Resources.Requests)
		}
		if len(spec.Resources.Limits) > 0 {
			fmt.Printf("  Limits:       %v\n", spec.Resources.Limits)
		}
	}

	fmt.Print("\nProceed with deployment? (yes/no): ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.ToLower(strings.TrimSpace(confirm))

	if confirm != "yes" && confirm != "y" {
		fmt.Println("Deployment cancelled.")
		return
	}

	// Deploy
	err := mcpDeployer.DeployMCPServer(context.Background(), spec)
	if err != nil {
		fmt.Printf("Error deploying server: %v\n", err)
		return
	}

	fmt.Printf("\n✓ MCP server '%s' deployed successfully in namespace '%s'!\n", spec.Name, spec.Namespace)
}

func deleteServer(mcpDeployer *deployer.SimpleDeployer, reader *bufio.Reader) {
	fmt.Println("\n=== Delete MCP Server ===\n")

	// Namespace
	fmt.Print("Enter namespace (default): ")
	namespace, _ := reader.ReadString('\n')
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		namespace = "default"
	}

	// List servers first to help user choose
	servers, err := mcpDeployer.ListMCPServers(context.Background(), namespace)
	if err != nil {
		fmt.Printf("Error listing servers: %v\n", err)
		return
	}

	if len(servers) == 0 {
		fmt.Printf("\nNo MCP servers found in namespace '%s'\n", namespace)
		return
	}

	fmt.Printf("\nAvailable MCP servers in namespace '%s':\n", namespace)
	for i, server := range servers {
		status := "unavailable"
		if server.Available {
			status = "available"
		}
		fmt.Printf("  %d. %s (%s) - %s\n", i+1, server.Name, server.Image, status)
	}

	// Name
	fmt.Print("\nEnter MCP server name to delete: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		fmt.Println("Error: Name is required")
		return
	}

	// Confirm deletion
	fmt.Printf("\n⚠️  WARNING: This will permanently delete the MCP server '%s' in namespace '%s'\n", name, namespace)
	fmt.Printf("This includes the Deployment and Service resources.\n")
	fmt.Print("\nAre you sure you want to proceed? (yes/no): ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.ToLower(strings.TrimSpace(confirm))

	if confirm != "yes" && confirm != "y" {
		fmt.Println("Deletion cancelled.")
		return
	}

	// Delete
	err = mcpDeployer.DeleteMCPServer(context.Background(), namespace, name)
	if err != nil {
		fmt.Printf("Error deleting server: %v\n", err)
		return
	}

	fmt.Printf("\n✓ MCP server '%s' deleted successfully from namespace '%s'!\n", name, namespace)
}

func promptForEnvVars(reader *bufio.Reader) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	fmt.Print("\nAdd environment variables? (yes/no): ")
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response != "yes" && response != "y" {
		return envVars
	}

	fmt.Println("\nEntering environment variables (press Enter with empty name to finish):")
	for {
		fmt.Print("\nEnv var name: ")
		name, _ := reader.ReadString('\n')
		name = strings.TrimSpace(name)
		if name == "" {
			break
		}

		fmt.Print("Is this a simple value or from a secret? (value/secret): ")
		varType, _ := reader.ReadString('\n')
		varType = strings.ToLower(strings.TrimSpace(varType))

		var envVar corev1.EnvVar
		envVar.Name = name

		if varType == "secret" {
			fmt.Print("Secret name: ")
			secretName, _ := reader.ReadString('\n')
			secretName = strings.TrimSpace(secretName)

			fmt.Print("Secret key: ")
			secretKey, _ := reader.ReadString('\n')
			secretKey = strings.TrimSpace(secretKey)

			envVar.ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: secretKey,
				},
			}
		} else {
			fmt.Print("Value: ")
			value, _ := reader.ReadString('\n')
			envVar.Value = strings.TrimSpace(value)
		}

		envVars = append(envVars, envVar)
		fmt.Printf("✓ Added environment variable: %s\n", name)
	}

	return envVars
}

func promptForArgs(reader *bufio.Reader) []string {
	var args []string

	fmt.Print("\nAdd command-line arguments? (yes/no): ")
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response != "yes" && response != "y" {
		return args
	}

	fmt.Println("\nEntering arguments (press Enter with empty value to finish):")
	for {
		fmt.Print("Argument: ")
		arg, _ := reader.ReadString('\n')
		arg = strings.TrimSpace(arg)
		if arg == "" {
			break
		}
		args = append(args, arg)
		fmt.Printf("✓ Added argument: %s\n", arg)
	}

	return args
}

func promptForSecretMounts(reader *bufio.Reader) []deployer.SecretMount {
	var secretMounts []deployer.SecretMount

	fmt.Print("\nAdd secret mounts? (yes/no): ")
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response != "yes" && response != "y" {
		return secretMounts
	}

	fmt.Println("\nEntering secret mounts (press Enter with empty secret name to finish):")
	for {
		fmt.Print("\nSecret name: ")
		secretName, _ := reader.ReadString('\n')
		secretName = strings.TrimSpace(secretName)
		if secretName == "" {
			break
		}

		fmt.Print("Mount path: ")
		mountPath, _ := reader.ReadString('\n')
		mountPath = strings.TrimSpace(mountPath)
		if mountPath == "" {
			fmt.Println("Error: Mount path is required")
			continue
		}

		secretMounts = append(secretMounts, deployer.SecretMount{
			SecretName: secretName,
			MountPath:  mountPath,
		})
		fmt.Printf("✓ Added secret mount: %s -> %s\n", secretName, mountPath)
	}

	return secretMounts
}

func promptForKeyValuePairs(reader *bufio.Reader, pairType string) map[string]string {
	pairs := make(map[string]string)

	fmt.Printf("\nAdd %ss? (yes/no): ", pairType)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response != "yes" && response != "y" {
		return pairs
	}

	fmt.Printf("\nEntering %ss (press Enter with empty key to finish):\n", pairType)
	for {
		fmt.Printf("\n%s key: ", strings.Title(pairType))
		key, _ := reader.ReadString('\n')
		key = strings.TrimSpace(key)
		if key == "" {
			break
		}

		fmt.Printf("%s value: ", strings.Title(pairType))
		value, _ := reader.ReadString('\n')
		value = strings.TrimSpace(value)

		pairs[key] = value
		fmt.Printf("✓ Added %s: %s=%s\n", pairType, key, value)
	}

	return pairs
}

func promptForResources(reader *bufio.Reader) *corev1.ResourceRequirements {
	fmt.Print("\nSet resource limits and requests? (yes/no): ")
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response != "yes" && response != "y" {
		return nil
	}

	resources := &corev1.ResourceRequirements{
		Requests: make(corev1.ResourceList),
		Limits:   make(corev1.ResourceList),
	}

	// CPU Request
	fmt.Print("\nCPU request (e.g., '100m', '0.5', '1') [leave empty to skip]: ")
	cpuRequest, _ := reader.ReadString('\n')
	cpuRequest = strings.TrimSpace(cpuRequest)
	if cpuRequest != "" {
		quantity, err := parseResourceQuantity(cpuRequest)
		if err != nil {
			fmt.Printf("Warning: Invalid CPU request format, skipping: %v\n", err)
		} else {
			resources.Requests[corev1.ResourceCPU] = quantity
		}
	}

	// Memory Request
	fmt.Print("Memory request (e.g., '128Mi', '1Gi', '512M') [leave empty to skip]: ")
	memRequest, _ := reader.ReadString('\n')
	memRequest = strings.TrimSpace(memRequest)
	if memRequest != "" {
		quantity, err := parseResourceQuantity(memRequest)
		if err != nil {
			fmt.Printf("Warning: Invalid memory request format, skipping: %v\n", err)
		} else {
			resources.Requests[corev1.ResourceMemory] = quantity
		}
	}

	// CPU Limit
	fmt.Print("CPU limit (e.g., '500m', '1', '2') [leave empty to skip]: ")
	cpuLimit, _ := reader.ReadString('\n')
	cpuLimit = strings.TrimSpace(cpuLimit)
	if cpuLimit != "" {
		quantity, err := parseResourceQuantity(cpuLimit)
		if err != nil {
			fmt.Printf("Warning: Invalid CPU limit format, skipping: %v\n", err)
		} else {
			resources.Limits[corev1.ResourceCPU] = quantity
		}
	}

	// Memory Limit
	fmt.Print("Memory limit (e.g., '256Mi', '2Gi', '1G') [leave empty to skip]: ")
	memLimit, _ := reader.ReadString('\n')
	memLimit = strings.TrimSpace(memLimit)
	if memLimit != "" {
		quantity, err := parseResourceQuantity(memLimit)
		if err != nil {
			fmt.Printf("Warning: Invalid memory limit format, skipping: %v\n", err)
		} else {
			resources.Limits[corev1.ResourceMemory] = quantity
		}
	}

	// If no resources were set, return nil
	if len(resources.Requests) == 0 && len(resources.Limits) == 0 {
		return nil
	}

	return resources
}

func parseResourceQuantity(value string) (resource.Quantity, error) {
	quantity, err := resource.ParseQuantity(value)
	if err != nil {
		return resource.Quantity{}, fmt.Errorf("invalid quantity: %w", err)
	}
	return quantity, nil
}

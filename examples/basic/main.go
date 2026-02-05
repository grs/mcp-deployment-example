package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/grs/mcp-deployment/pkg/deployer"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Build Kubernetes config from kubeconfig file
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Failed to build config: %v", err)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create clientset: %v", err)
	}

	// Create MCP deployer
	mcpDeployer := deployer.NewSimpleDeployer(clientset)

	// Example 1: Deploy an MCP server
	fmt.Println("Deploying MCP server...")
	spec := &deployer.MCPServerSpec{
		Name:      "example-mcp-server",
		Namespace: "default",
		Image:     "example/mcp-server:latest",
		Port:      8080,
		EnvVars: []corev1.EnvVar{
			{
				Name:  "LOG_LEVEL",
				Value: "info",
			},
			{
				Name: "API_KEY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "api-credentials",
						},
						Key: "key",
					},
				},
			},
		},
		Args: []string{"--enable-monitoring", "--config=/etc/mcp/config.yaml"},
		SecretMounts: []deployer.SecretMount{
			{
				SecretName: "mcp-config",
				MountPath:  "/etc/mcp",
			},
		},
		ServiceAccount: "mcp-server",
		Labels: map[string]string{
			"app":         "mcp-server",
			"environment": "production",
			"team":        "platform",
		},
		Annotations: map[string]string{
			"description":     "Example MCP server deployment",
			"contact":         "platform-team@example.com",
			"deployment-date": "2026-02-05",
		},
		Resources: &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		},
	}

	err = mcpDeployer.DeployMCPServer(context.Background(), spec)
	if err != nil {
		log.Fatalf("Failed to deploy MCP server: %v", err)
	}
	fmt.Println("MCP server deployed successfully!")

	// Example 2: List all MCP servers
	fmt.Println("\nListing MCP servers in namespace 'default'...")
	servers, err := mcpDeployer.ListMCPServers(context.Background(), "default")
	if err != nil {
		log.Fatalf("Failed to list MCP servers: %v", err)
	}

	fmt.Printf("\nFound %d MCP server(s):\n\n", len(servers))
	for i, server := range servers {
		fmt.Printf("Server %d:\n", i+1)
		fmt.Printf("  Name: %s\n", server.Name)
		fmt.Printf("  Namespace: %s\n", server.Namespace)
		fmt.Printf("  Image: %s\n", server.Image)
		fmt.Printf("  Available: %t\n", server.Available)

		fmt.Println("  Labels:")
		for k, v := range server.Labels {
			fmt.Printf("    %s: %s\n", k, v)
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

	// Example 3: Delete an MCP server (commented out by default)
	// Uncomment to test deletion
	/*
		fmt.Println("\nDeleting MCP server 'example-mcp-server'...")
		err = mcpDeployer.DeleteMCPServer(context.Background(), "default", "example-mcp-server")
		if err != nil {
			log.Fatalf("Failed to delete MCP server: %v", err)
		}
		fmt.Println("MCP server deleted successfully!")
	*/
}

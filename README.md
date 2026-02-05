# MCP Deployment

A Go library for deploying and managing MCP (Model Context Protocol) servers on Kubernetes.

## Features

- **Interactive CLI Wizard** for easy MCP server deployment
- Abstract interface for MCP server deployment
- Kubernetes implementation with Deployment and Service creation
- Automatic labeling with `mcp.opendatahub.io/mcp-server` label
- Support for:
  - Custom images and ports
  - Environment variables (simple values or secret references)
  - Command-line arguments
  - Secret mounts
  - Service accounts
  - Resource limits and requests (CPU and memory)
  - Custom labels and annotations

## Installation

### Install the library

```bash
go get github.com/grs/mcp-deployment
```

### Build the CLI wizard

```bash
cd cmd/wizard
go build -o wizard
```

Or from the project root:

```bash
go build -o wizard ./cmd/wizard
```

## Project Structure

```
.
├── cmd/
│   └── wizard/            # Interactive CLI tool
│       └── main.go
├── pkg/
│   └── deployer/          # Main library package
│       ├── deployer.go    # Interface and type definitions
│       └── simple_deployer.go # Simple Kubernetes implementation
├── examples/
│   └── basic/             # Basic usage example
│       └── main.go
├── go.mod
├── go.sum
├── README.md
├── CONTRIBUTING.md
└── .gitignore
```

## Usage

### Using the CLI Wizard (Recommended for Getting Started)

The wizard provides an interactive command-line interface for deploying and managing MCP servers:

```bash
./wizard
```

The wizard will present you with a menu:

```
=== MCP Server Deployment Wizard ===
1. List MCP servers
2. Deploy new MCP server
3. Delete MCP server
4. Exit

Select an option:
```

**Listing MCP Servers**: Select option 1 and enter the namespace to see all deployed MCP servers.

**Deploying a New MCP Server**: Select option 2 and the wizard will interactively prompt you for:
- Server name and namespace
- Container image and port
- Environment variables (with option for simple values or secret references)
- Command-line arguments
- Secret mounts (volume mounts for secrets)
- Service account
- Labels and annotations
- Resource limits and requests (CPU and memory)

For environment variables, the wizard asks whether each variable should be:
- **value**: A simple string value
- **secret**: A reference to a Kubernetes secret (you'll provide secret name and key)

**Deleting an MCP Server**: Select option 3 to delete an existing MCP server. The wizard will:
- List all available MCP servers in the namespace
- Prompt for the server name to delete
- Show a warning and ask for confirmation before proceeding

### Programmatic Usage

For programmatic use in your Go applications:

#### Creating a Deployer

```go
import (
    "github.com/grs/mcp-deployment/pkg/deployer"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

// Create Kubernetes client
config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
if err != nil {
    // handle error
}
clientset, err := kubernetes.NewForConfig(config)
if err != nil {
    // handle error
}

// Create MCP deployer
mcpDeployer := deployer.NewSimpleDeployer(clientset)
```

#### Deploying an MCP Server

```go
import (
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
)

spec := &deployer.MCPServerSpec{
    Name:      "my-mcp-server",
    Namespace: "default",
    Image:     "my-mcp-image:latest",
    Port:      8080,
    EnvVars: []corev1.EnvVar{
        {
            Name:  "API_KEY",
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
    Args: []string{"--verbose", "--config=/etc/config.yaml"},
    SecretMounts: []deployer.SecretMount{
        {
            SecretName: "mcp-config",
            MountPath:  "/etc/config",
        },
    },
    ServiceAccount: "mcp-server-sa",
    Labels: map[string]string{
        "app": "my-mcp-server",
        "environment": "production",
    },
    Annotations: map[string]string{
        "description": "Production MCP server",
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

err := mcpDeployer.DeployMCPServer(context.Background(), spec)
if err != nil {
    // handle error
}
```

#### Listing MCP Servers

```go
servers, err := mcpDeployer.ListMCPServers(context.Background(), "default")
if err != nil {
    // handle error
}

for _, server := range servers {
    fmt.Printf("Name: %s\n", server.Name)
    fmt.Printf("Namespace: %s\n", server.Namespace)
    fmt.Printf("Image: %s\n", server.Image)
    fmt.Printf("Available: %t\n", server.Available)
    fmt.Println("Conditions:")
    for _, condition := range server.Conditions {
        fmt.Printf("  - %s\n", condition)
    }
}
```

#### Deleting an MCP Server

```go
err := mcpDeployer.DeleteMCPServer(context.Background(), "default", "my-mcp-server")
if err != nil {
    // handle error
}
```

## Interface

The `MCPDeployer` interface provides three main methods:

```go
type MCPDeployer interface {
    // DeployMCPServer creates a Deployment and Service for an MCP server
    DeployMCPServer(ctx context.Context, spec *MCPServerSpec) error

    // ListMCPServers lists all MCP servers in the specified namespace
    ListMCPServers(ctx context.Context, namespace string) ([]MCPServerStatus, error)

    // DeleteMCPServer deletes an MCP server (Deployment and Service) by name
    DeleteMCPServer(ctx context.Context, namespace, name string) error
}
```

## Automatic Labeling

All deployed MCP servers are automatically labeled with `mcp.opendatahub.io/mcp-server=true` in addition to any custom labels you provide. This label is used to identify and list MCP server deployments.

package deployer

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

// SecretMount represents a secret to be mounted in the MCP server pod
type SecretMount struct {
	SecretName string
	MountPath  string
}

// MCPServerSpec contains the specification for deploying an MCP server
type MCPServerSpec struct {
	Name             string
	Namespace        string
	Image            string
	Port             int32
	EnvVars          []corev1.EnvVar
	Args             []string
	SecretMounts     []SecretMount
	ServiceAccount   string
	Labels           map[string]string
	Annotations      map[string]string
	Resources        *corev1.ResourceRequirements
}

// MCPServerStatus represents the status of a deployed MCP server
type MCPServerStatus struct {
	Name        string
	Namespace   string
	Image       string
	Available   bool
	Endpoint    string
	Labels      map[string]string
	Annotations map[string]string
	Conditions  []string
}

// MCPDeployer is the interface for managing MCP server deployments
type MCPDeployer interface {
	// DeployMCPServer creates a Deployment and Service for an MCP server
	DeployMCPServer(ctx context.Context, spec *MCPServerSpec) error

	// ListMCPServers lists all MCP servers in the specified namespace
	ListMCPServers(ctx context.Context, namespace string) ([]MCPServerStatus, error)

	// DeleteMCPServer deletes an MCP server (Deployment and Service) by name
	DeleteMCPServer(ctx context.Context, namespace, name string) error
}

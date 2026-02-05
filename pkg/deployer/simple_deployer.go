package deployer

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

const (
	// MCPServerLabel is the label used to identify MCP server deployments
	MCPServerLabel = "mcp.opendatahub.io/mcp-server"
)

// SimpleDeployer implements the MCPDeployer interface using Kubernetes client
type SimpleDeployer struct {
	clientset *kubernetes.Clientset
}

// NewSimpleDeployer creates a new SimpleDeployer instance
func NewSimpleDeployer(clientset *kubernetes.Clientset) *SimpleDeployer {
	return &SimpleDeployer{
		clientset: clientset,
	}
}

// DeployMCPServer creates a Deployment and Service for an MCP server
func (d *SimpleDeployer) DeployMCPServer(ctx context.Context, spec *MCPServerSpec) error {
	if err := d.createDeployment(ctx, spec); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	if err := d.createService(ctx, spec); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

// ListMCPServers lists all MCP servers in the specified namespace
func (d *SimpleDeployer) ListMCPServers(ctx context.Context, namespace string) ([]MCPServerStatus, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=true", MCPServerLabel),
	}

	deployments, err := d.clientset.AppsV1().Deployments(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	var servers []MCPServerStatus
	for _, deployment := range deployments.Items {
		status := MCPServerStatus{
			Name:        deployment.Name,
			Namespace:   deployment.Namespace,
			Available:   deployment.Status.AvailableReplicas > 0,
			Labels:      deployment.Labels,
			Annotations: deployment.Annotations,
		}

		// Extract image from the first container
		if len(deployment.Spec.Template.Spec.Containers) > 0 {
			status.Image = deployment.Spec.Template.Spec.Containers[0].Image
		}

		// Get the service to extract endpoint (only if deployment is available)
		if status.Available {
			service, err := d.clientset.CoreV1().Services(namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
			if err == nil && len(service.Spec.Ports) > 0 {
				status.Endpoint = fmt.Sprintf("%s:%d", service.Name, service.Spec.Ports[0].Port)
			}
		}

		// Extract condition messages
		for _, condition := range deployment.Status.Conditions {
			status.Conditions = append(status.Conditions,
				fmt.Sprintf("%s: %s - %s", condition.Type, condition.Status, condition.Message))
		}

		servers = append(servers, status)
	}

	return servers, nil
}

// createDeployment creates a Kubernetes Deployment for the MCP server
func (d *SimpleDeployer) createDeployment(ctx context.Context, spec *MCPServerSpec) error {
	labels := d.mergeLabels(spec.Labels)

	replicas := int32(1)

	// Build volumes and volume mounts from secret mounts
	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount
	for i, secretMount := range spec.SecretMounts {
		volumeName := fmt.Sprintf("secret-%d", i)
		volumes = append(volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretMount.SecretName,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: secretMount.MountPath,
			ReadOnly:  true,
		})
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        spec.Name,
			Namespace:   spec.Namespace,
			Labels:      labels,
			Annotations: spec.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: spec.Annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: spec.ServiceAccount,
					Containers: []corev1.Container{
						{
							Name:  "mcp-server",
							Image: spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "mcp",
									ContainerPort: spec.Port,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env:          spec.EnvVars,
							Args:         spec.Args,
							VolumeMounts: volumeMounts,
							Resources:    d.getResources(spec.Resources),
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	_, err := d.clientset.AppsV1().Deployments(spec.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	return nil
}

// createService creates a Kubernetes Service for the MCP server
func (d *SimpleDeployer) createService(ctx context.Context, spec *MCPServerSpec) error {
	labels := d.mergeLabels(spec.Labels)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        spec.Name,
			Namespace:   spec.Namespace,
			Labels:      labels,
			Annotations: spec.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "mcp",
					Port:       spec.Port,
					TargetPort: intstr.FromInt(int(spec.Port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	_, err := d.clientset.CoreV1().Services(spec.Namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

// DeleteMCPServer deletes an MCP server (Deployment and Service) by name
func (d *SimpleDeployer) DeleteMCPServer(ctx context.Context, namespace, name string) error {
	// Delete the deployment
	err := d.clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	// Delete the service
	err = d.clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	return nil
}

// getResources returns the resource requirements or an empty one if nil
func (d *SimpleDeployer) getResources(resources *corev1.ResourceRequirements) corev1.ResourceRequirements {
	if resources != nil {
		return *resources
	}
	return corev1.ResourceRequirements{}
}

// mergeLabels merges user-provided labels with the required MCP server label
func (d *SimpleDeployer) mergeLabels(userLabels map[string]string) map[string]string {
	labels := make(map[string]string)

	// Copy user labels
	for k, v := range userLabels {
		labels[k] = v
	}

	// Add the required MCP server label
	labels[MCPServerLabel] = "true"

	return labels
}

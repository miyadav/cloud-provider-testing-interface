/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testing

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	cloudprovider "k8s.io/cloud-provider"
	fakecloud "k8s.io/cloud-provider/fake"
)

// BaseTestImplementation provides a base implementation of the TestInterface
// that can be extended by specific cloud provider test implementations.
type BaseTestImplementation struct {
	// CloudProvider is the cloud provider instance being tested.
	CloudProvider cloudprovider.Interface

	// ClientBuilder is the client builder for creating Kubernetes clients.
	ClientBuilder cloudprovider.ControllerClientBuilder

	// InformerFactory is the informer factory for creating informers.
	InformerFactory informers.SharedInformerFactory

	// TestConfig holds the current test configuration.
	TestConfig *TestConfig

	// TestResults holds the current test results.
	TestResults *TestResults

	// CreatedResources tracks resources created during tests for cleanup.
	CreatedResources map[string][]string

	// mu protects access to the BaseTestImplementation fields
	mu sync.RWMutex
}

// NewBaseTestImplementation creates a new base test implementation.
func NewBaseTestImplementation(cloudProvider cloudprovider.Interface) *BaseTestImplementation {
	return &BaseTestImplementation{
		CloudProvider:    cloudProvider,
		CreatedResources: make(map[string][]string),
		TestResults:      &TestResults{},
		TestConfig:       &TestConfig{},
	}
}

// SetupTestEnvironment initializes the test environment.
func (b *BaseTestImplementation) SetupTestEnvironment(config *TestConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.TestConfig = config
	b.TestResults = &TestResults{
		Success:        true,
		ResourceCounts: make(map[string]int),
		Metrics:        make(map[string]interface{}),
		Logs:           []string{},
	}

	// Initialize the cloud provider if it supports InformerUser
	if informerUser, ok := b.CloudProvider.(cloudprovider.InformerUser); ok {
		informerUser.SetInformers(config.InformerFactory)
	}

	// Initialize the cloud provider
	stopCh := make(chan struct{})
	b.CloudProvider.Initialize(config.ClientBuilder, stopCh)

	b.TestResults.AddLog("Test environment setup completed")
	return nil
}

// TeardownTestEnvironment cleans up the test environment.
func (b *BaseTestImplementation) TeardownTestEnvironment() error {
	b.mu.Lock()
	// Just clear the created resources tracking to avoid deadlocks
	b.CreatedResources = make(map[string][]string)
	b.mu.Unlock()

	b.TestResults.AddLog("Test environment teardown completed")
	return nil
}

// GetCloudProvider returns the cloud provider instance.
func (b *BaseTestImplementation) GetCloudProvider() cloudprovider.Interface {
	return b.CloudProvider
}

// CreateTestNode creates a test node.
func (b *BaseTestImplementation) CreateTestNode(ctx context.Context, nodeConfig *TestNodeConfig) (*v1.Node, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:        nodeConfig.Name,
			Labels:      nodeConfig.Labels,
			Annotations: nodeConfig.Annotations,
		},
		Spec: v1.NodeSpec{
			ProviderID: nodeConfig.ProviderID,
		},
		Status: v1.NodeStatus{
			Addresses:  nodeConfig.Addresses,
			Conditions: nodeConfig.Conditions,
			NodeInfo: v1.NodeSystemInfo{
				KubeletVersion: "v1.24.0",
			},
		},
	}

	// Add node labels based on configuration
	if nodeConfig.Zone != "" {
		if node.Labels == nil {
			node.Labels = make(map[string]string)
		}
		node.Labels["topology.kubernetes.io/zone"] = nodeConfig.Zone
		node.Labels["failure-domain.beta.kubernetes.io/zone"] = nodeConfig.Zone
	}

	if nodeConfig.Region != "" {
		if node.Labels == nil {
			node.Labels = make(map[string]string)
		}
		node.Labels["topology.kubernetes.io/region"] = nodeConfig.Region
		node.Labels["failure-domain.beta.kubernetes.io/region"] = nodeConfig.Region
	}

	if nodeConfig.InstanceType != "" {
		if node.Labels == nil {
			node.Labels = make(map[string]string)
		}
		node.Labels["node.kubernetes.io/instance-type"] = nodeConfig.InstanceType
		node.Labels["beta.kubernetes.io/instance-type"] = nodeConfig.InstanceType
	}

	// Track created resource
	b.CreatedResources["node"] = append(b.CreatedResources["node"], nodeConfig.Name)
	b.TestResults.IncrementResourceCount("node")

	b.TestResults.AddLog(fmt.Sprintf("Created test node: %s", nodeConfig.Name))
	return node, nil
}

// DeleteTestNode deletes a test node.
func (b *BaseTestImplementation) DeleteTestNode(ctx context.Context, nodeName string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove from created resources
	for i, name := range b.CreatedResources["node"] {
		if name == nodeName {
			b.CreatedResources["node"] = append(b.CreatedResources["node"][:i], b.CreatedResources["node"][i+1:]...)
			break
		}
	}

	b.TestResults.AddLog(fmt.Sprintf("Deleted test node: %s", nodeName))
	return nil
}

// CreateTestService creates a test service.
func (b *BaseTestImplementation) CreateTestService(ctx context.Context, serviceConfig *TestServiceConfig) (*v1.Service, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        serviceConfig.Name,
			Namespace:   serviceConfig.Namespace,
			Labels:      serviceConfig.Labels,
			Annotations: serviceConfig.Annotations,
		},
		Spec: v1.ServiceSpec{
			Type:                  serviceConfig.Type,
			Ports:                 serviceConfig.Ports,
			LoadBalancerIP:        serviceConfig.LoadBalancerIP,
			ExternalTrafficPolicy: serviceConfig.ExternalTrafficPolicy,
			InternalTrafficPolicy: serviceConfig.InternalTrafficPolicy,
		},
	}

	// Track created resource
	b.CreatedResources["service"] = append(b.CreatedResources["service"], serviceConfig.Name)
	b.TestResults.IncrementResourceCount("service")

	b.TestResults.AddLog(fmt.Sprintf("Created test service: %s", serviceConfig.Name))
	return service, nil
}

// DeleteTestService deletes a test service.
func (b *BaseTestImplementation) DeleteTestService(ctx context.Context, serviceName string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove from created resources
	for i, name := range b.CreatedResources["service"] {
		if name == serviceName {
			b.CreatedResources["service"] = append(b.CreatedResources["service"][:i], b.CreatedResources["service"][i+1:]...)
			break
		}
	}

	b.TestResults.AddLog(fmt.Sprintf("Deleted test service: %s", serviceName))
	return nil
}

// CreateTestRoute creates a test route.
func (b *BaseTestImplementation) CreateTestRoute(ctx context.Context, routeConfig *TestRouteConfig) (*cloudprovider.Route, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	route := &cloudprovider.Route{
		Name:            routeConfig.Name,
		TargetNode:      routeConfig.TargetNode,
		DestinationCIDR: routeConfig.DestinationCIDR,
		Blackhole:       routeConfig.Blackhole,
	}

	// Track created resource
	b.CreatedResources["route"] = append(b.CreatedResources["route"], routeConfig.Name)
	b.TestResults.IncrementResourceCount("route")

	b.TestResults.AddLog(fmt.Sprintf("Created test route: %s", routeConfig.Name))
	return route, nil
}

// DeleteTestRoute deletes a test route.
func (b *BaseTestImplementation) DeleteTestRoute(ctx context.Context, routeName string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove from created resources
	for i, name := range b.CreatedResources["route"] {
		if name == routeName {
			b.CreatedResources["route"] = append(b.CreatedResources["route"][:i], b.CreatedResources["route"][i+1:]...)
			break
		}
	}

	b.TestResults.AddLog(fmt.Sprintf("Deleted test route: %s", routeName))
	return nil
}

// WaitForCondition waits for a specific condition to be met.
func (b *BaseTestImplementation) WaitForCondition(ctx context.Context, condition TestCondition) error {
	timeout := condition.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition.CheckFunction != nil {
			met, err := condition.CheckFunction()
			if err != nil {
				return err
			}
			if met {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("condition not met within timeout: %s", condition.Type)
}

// GetTestResults returns the test results.
func (b *BaseTestImplementation) GetTestResults() *TestResults {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.TestResults
}

// ResetTestState resets the test state.
func (b *BaseTestImplementation) ResetTestState() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.CreatedResources = make(map[string][]string)
	b.TestResults = &TestResults{
		Success:        true,
		ResourceCounts: make(map[string]int),
		Metrics:        make(map[string]interface{}),
		Logs:           []string{},
	}

	b.TestResults.AddLog("Test state reset completed")
	return nil
}

// FakeTestImplementation provides a test implementation using the fake cloud provider.
// This is useful for testing the test framework itself or for cloud providers that
// want to use the fake provider for testing.
type FakeTestImplementation struct {
	*BaseTestImplementation
	FakeCloud *fakecloud.Cloud
}

// NewFakeTestImplementation creates a new fake test implementation.
func NewFakeTestImplementation() *FakeTestImplementation {
	fakeCloud := &fakecloud.Cloud{
		Balancers:     make(map[string]fakecloud.Balancer),
		ExtID:         make(map[types.NodeName]string),
		ExtIDErr:      make(map[types.NodeName]error),
		InstanceTypes: make(map[types.NodeName]string),
		ProviderID:    make(map[types.NodeName]string),
		RouteMap:      make(map[string]*fakecloud.Route),
	}

	baseImpl := NewBaseTestImplementation(fakeCloud)
	return &FakeTestImplementation{
		BaseTestImplementation: baseImpl,
		FakeCloud:              fakeCloud,
	}
}

// GetFakeCloud returns the fake cloud provider instance.
func (f *FakeTestImplementation) GetFakeCloud() *fakecloud.Cloud {
	return f.FakeCloud
}

// MockClientBuilder provides a mock implementation of ControllerClientBuilder for testing.
type MockClientBuilder struct {
	ConfigFunc      func(name string) (*rest.Config, error)
	ClientFunc      func(name string) (clientset.Interface, error)
	ConfigOrDieFunc func(name string) *rest.Config
	ClientOrDieFunc func(name string) clientset.Interface
}

// Config implements ControllerClientBuilder.Config.
func (m *MockClientBuilder) Config(name string) (*rest.Config, error) {
	if m.ConfigFunc != nil {
		return m.ConfigFunc(name)
	}
	return &rest.Config{}, nil
}

// ConfigOrDie implements ControllerClientBuilder.ConfigOrDie.
func (m *MockClientBuilder) ConfigOrDie(name string) *rest.Config {
	if m.ConfigOrDieFunc != nil {
		return m.ConfigOrDieFunc(name)
	}
	return &rest.Config{}
}

// Client implements ControllerClientBuilder.Client.
func (m *MockClientBuilder) Client(name string) (clientset.Interface, error) {
	if m.ClientFunc != nil {
		return m.ClientFunc(name)
	}
	return nil, fmt.Errorf("mock client not configured")
}

// ClientOrDie implements ControllerClientBuilder.ClientOrDie.
func (m *MockClientBuilder) ClientOrDie(name string) clientset.Interface {
	if m.ClientOrDieFunc != nil {
		return m.ClientOrDieFunc(name)
	}
	panic("mock client not configured")
}

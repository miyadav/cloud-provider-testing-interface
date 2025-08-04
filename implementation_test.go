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
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	fakecloud "k8s.io/cloud-provider/fake"
)

// TestNewBaseTestImplementation tests creating a new base test implementation
func TestNewBaseTestImplementation(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	if baseImpl.CloudProvider != fakeCloud {
		t.Error("Expected cloud provider to be set")
	}

	if baseImpl.CreatedResources == nil {
		t.Error("Expected created resources map to be initialized")
	}

	if baseImpl.TestResults == nil {
		t.Error("Expected test results to be initialized")
	}

	if baseImpl.TestConfig == nil {
		t.Error("Expected test config to be initialized")
	}
}

// TestBaseTestImplementationSetupTestEnvironment tests setting up the test environment
func TestBaseTestImplementationSetupTestEnvironment(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	config := &TestConfig{
		ProviderName:         "test-provider",
		ClusterName:          "test-cluster",
		Region:               "us-west-1",
		Zone:                 "us-west-1a",
		TestTimeout:          5 * time.Minute,
		CleanupResources:     true,
		MockExternalServices: true,
		ClientBuilder:        &MockClientBuilder{},
		InformerFactory:      informers.NewSharedInformerFactory(fake.NewSimpleClientset(), 0),
	}

	err := baseImpl.SetupTestEnvironment(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if baseImpl.TestConfig != config {
		t.Error("Expected test config to be set")
	}

	if baseImpl.TestResults == nil {
		t.Error("Expected test results to be initialized")
	}

	if !baseImpl.TestResults.Success {
		t.Error("Expected test results success to be true")
	}

	if baseImpl.TestResults.ResourceCounts == nil {
		t.Error("Expected resource counts to be initialized")
	}

	if baseImpl.TestResults.Metrics == nil {
		t.Error("Expected metrics to be initialized")
	}

	if baseImpl.TestResults.Logs == nil {
		t.Error("Expected logs to be initialized")
	}
}

// TestBaseTestImplementationTeardownTestEnvironment tests tearing down the test environment
func TestBaseTestImplementationTeardownTestEnvironment(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	// Add some created resources
	baseImpl.CreatedResources["node"] = []string{"test-node-1", "test-node-2"}
	baseImpl.CreatedResources["service"] = []string{"test-service-1"}

	err := baseImpl.TeardownTestEnvironment()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify resources are cleared
	if len(baseImpl.CreatedResources) != 0 {
		t.Error("Expected created resources to be cleared")
	}
}

// TestBaseTestImplementationGetCloudProvider tests getting the cloud provider
func TestBaseTestImplementationGetCloudProvider(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	cloud := baseImpl.GetCloudProvider()
	if cloud != fakeCloud {
		t.Error("Expected cloud provider to be returned")
	}
}

// TestBaseTestImplementationCreateTestNode tests creating a test node
func TestBaseTestImplementationCreateTestNode(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	nodeConfig := &TestNodeConfig{
		Name:         "test-node",
		ProviderID:   "test://test-node",
		InstanceType: "t3.medium",
		Zone:         "us-west-1a",
		Region:       "us-west-1",
		Addresses: []v1.NodeAddress{
			{Type: v1.NodeInternalIP, Address: "10.0.0.1"},
			{Type: v1.NodeExternalIP, Address: "192.168.1.1"},
		},
		Labels: map[string]string{
			"node-role.kubernetes.io/worker": "true",
		},
		Annotations: map[string]string{
			"test.annotation": "test-value",
		},
		Conditions: []v1.NodeCondition{
			{Type: v1.NodeReady, Status: v1.ConditionTrue},
		},
	}

	ctx := context.Background()
	node, err := baseImpl.CreateTestNode(ctx, nodeConfig)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if node.Name != "test-node" {
		t.Errorf("Expected node name 'test-node', got '%s'", node.Name)
	}

	if node.Spec.ProviderID != "test://test-node" {
		t.Errorf("Expected provider ID 'test://test-node', got '%s'", node.Spec.ProviderID)
	}

	if len(node.Status.Addresses) != 2 {
		t.Errorf("Expected 2 addresses, got %d", len(node.Status.Addresses))
	}

	if len(node.Labels) == 0 {
		t.Error("Expected node labels to be set")
	}

	if len(node.Annotations) == 0 {
		t.Error("Expected node annotations to be set")
	}

	if len(node.Status.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(node.Status.Conditions))
	}

	// Verify zone and region labels are set
	if node.Labels["topology.kubernetes.io/zone"] != "us-west-1a" {
		t.Errorf("Expected zone label 'us-west-1a', got '%s'", node.Labels["topology.kubernetes.io/zone"])
	}

	if node.Labels["topology.kubernetes.io/region"] != "us-west-1" {
		t.Errorf("Expected region label 'us-west-1', got '%s'", node.Labels["topology.kubernetes.io/region"])
	}

	if node.Labels["node.kubernetes.io/instance-type"] != "t3.medium" {
		t.Errorf("Expected instance type label 't3.medium', got '%s'", node.Labels["node.kubernetes.io/instance-type"])
	}

	// Verify resource tracking
	if len(baseImpl.CreatedResources["node"]) != 1 {
		t.Errorf("Expected 1 created node, got %d", len(baseImpl.CreatedResources["node"]))
	}

	if baseImpl.CreatedResources["node"][0] != "test-node" {
		t.Errorf("Expected created node 'test-node', got '%s'", baseImpl.CreatedResources["node"][0])
	}

	if baseImpl.TestResults.ResourceCounts["node"] != 1 {
		t.Errorf("Expected node count 1, got %d", baseImpl.TestResults.ResourceCounts["node"])
	}
}

// TestBaseTestImplementationDeleteTestNode tests deleting a test node
func TestBaseTestImplementationDeleteTestNode(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	// Add a node to created resources
	baseImpl.CreatedResources["node"] = []string{"test-node-1", "test-node-2"}

	ctx := context.Background()
	err := baseImpl.DeleteTestNode(ctx, "test-node-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify node is removed from created resources
	if len(baseImpl.CreatedResources["node"]) != 1 {
		t.Errorf("Expected 1 remaining node, got %d", len(baseImpl.CreatedResources["node"]))
	}

	if baseImpl.CreatedResources["node"][0] != "test-node-2" {
		t.Errorf("Expected remaining node 'test-node-2', got '%s'", baseImpl.CreatedResources["node"][0])
	}
}

// TestBaseTestImplementationCreateTestService tests creating a test service
func TestBaseTestImplementationCreateTestService(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	internalTrafficPolicy := v1.ServiceInternalTrafficPolicyCluster
	serviceConfig := &TestServiceConfig{
		Name:      "test-service",
		Namespace: "default",
		Type:      v1.ServiceTypeLoadBalancer,
		Ports: []v1.ServicePort{
			{Port: 80, TargetPort: intstr.FromInt(8080), Protocol: v1.ProtocolTCP},
			{Port: 443, TargetPort: intstr.FromInt(8443), Protocol: v1.ProtocolTCP},
		},
		LoadBalancerIP:        "192.168.1.100",
		ExternalTrafficPolicy: v1.ServiceExternalTrafficPolicyCluster,
		InternalTrafficPolicy: &internalTrafficPolicy,
		Labels: map[string]string{
			"app": "test-app",
		},
		Annotations: map[string]string{
			"service.annotation": "test-value",
		},
	}

	ctx := context.Background()
	service, err := baseImpl.CreateTestService(ctx, serviceConfig)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if service.Name != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", service.Name)
	}

	if service.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", service.Namespace)
	}

	if service.Spec.Type != v1.ServiceTypeLoadBalancer {
		t.Errorf("Expected service type LoadBalancer, got %v", service.Spec.Type)
	}

	if len(service.Spec.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(service.Spec.Ports))
	}

	if service.Spec.LoadBalancerIP != "192.168.1.100" {
		t.Errorf("Expected load balancer IP '192.168.1.100', got '%s'", service.Spec.LoadBalancerIP)
	}

	if service.Spec.ExternalTrafficPolicy != v1.ServiceExternalTrafficPolicyCluster {
		t.Errorf("Expected external traffic policy Cluster, got %v", service.Spec.ExternalTrafficPolicy)
	}

	if *service.Spec.InternalTrafficPolicy != v1.ServiceInternalTrafficPolicyCluster {
		t.Errorf("Expected internal traffic policy Cluster, got %v", *service.Spec.InternalTrafficPolicy)
	}

	if len(service.Labels) == 0 {
		t.Error("Expected service labels to be set")
	}

	if len(service.Annotations) == 0 {
		t.Error("Expected service annotations to be set")
	}

	// Verify resource tracking
	if len(baseImpl.CreatedResources["service"]) != 1 {
		t.Errorf("Expected 1 created service, got %d", len(baseImpl.CreatedResources["service"]))
	}

	if baseImpl.CreatedResources["service"][0] != "test-service" {
		t.Errorf("Expected created service 'test-service', got '%s'", baseImpl.CreatedResources["service"][0])
	}

	if baseImpl.TestResults.ResourceCounts["service"] != 1 {
		t.Errorf("Expected service count 1, got %d", baseImpl.TestResults.ResourceCounts["service"])
	}
}

// TestBaseTestImplementationDeleteTestService tests deleting a test service
func TestBaseTestImplementationDeleteTestService(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	// Add a service to created resources
	baseImpl.CreatedResources["service"] = []string{"test-service-1", "test-service-2"}

	ctx := context.Background()
	err := baseImpl.DeleteTestService(ctx, "test-service-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify service is removed from created resources
	if len(baseImpl.CreatedResources["service"]) != 1 {
		t.Errorf("Expected 1 remaining service, got %d", len(baseImpl.CreatedResources["service"]))
	}

	if baseImpl.CreatedResources["service"][0] != "test-service-2" {
		t.Errorf("Expected remaining service 'test-service-2', got '%s'", baseImpl.CreatedResources["service"][0])
	}
}

// TestBaseTestImplementationCreateTestRoute tests creating a test route
func TestBaseTestImplementationCreateTestRoute(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	routeConfig := &TestRouteConfig{
		Name:            "test-route",
		ClusterName:     "test-cluster",
		TargetNode:      types.NodeName("test-node"),
		DestinationCIDR: "10.100.0.0/24",
		Blackhole:       false,
	}

	ctx := context.Background()
	route, err := baseImpl.CreateTestRoute(ctx, routeConfig)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if route.Name != "test-route" {
		t.Errorf("Expected route name 'test-route', got '%s'", route.Name)
	}

	if route.TargetNode != types.NodeName("test-node") {
		t.Errorf("Expected target node 'test-node', got '%s'", route.TargetNode)
	}

	if route.DestinationCIDR != "10.100.0.0/24" {
		t.Errorf("Expected destination CIDR '10.100.0.0/24', got '%s'", route.DestinationCIDR)
	}

	if route.Blackhole {
		t.Error("Expected blackhole to be false")
	}

	// Verify resource tracking
	if len(baseImpl.CreatedResources["route"]) != 1 {
		t.Errorf("Expected 1 created route, got %d", len(baseImpl.CreatedResources["route"]))
	}

	if baseImpl.CreatedResources["route"][0] != "test-route" {
		t.Errorf("Expected created route 'test-route', got '%s'", baseImpl.CreatedResources["route"][0])
	}

	if baseImpl.TestResults.ResourceCounts["route"] != 1 {
		t.Errorf("Expected route count 1, got %d", baseImpl.TestResults.ResourceCounts["route"])
	}
}

// TestBaseTestImplementationDeleteTestRoute tests deleting a test route
func TestBaseTestImplementationDeleteTestRoute(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	// Add a route to created resources
	baseImpl.CreatedResources["route"] = []string{"test-route-1", "test-route-2"}

	ctx := context.Background()
	err := baseImpl.DeleteTestRoute(ctx, "test-route-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify route is removed from created resources
	if len(baseImpl.CreatedResources["route"]) != 1 {
		t.Errorf("Expected 1 remaining route, got %d", len(baseImpl.CreatedResources["route"]))
	}

	if baseImpl.CreatedResources["route"][0] != "test-route-2" {
		t.Errorf("Expected remaining route 'test-route-2', got '%s'", baseImpl.CreatedResources["route"][0])
	}
}

// TestBaseTestImplementationWaitForCondition tests waiting for a condition
func TestBaseTestImplementationWaitForCondition(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	// Test condition that is immediately met
	condition := TestCondition{
		Type:    "Ready",
		Status:  "True",
		Timeout: 5 * time.Second,
		CheckFunction: func() (bool, error) {
			return true, nil
		},
	}

	ctx := context.Background()
	err := baseImpl.WaitForCondition(ctx, condition)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test condition that times out
	timeoutCondition := TestCondition{
		Type:    "Timeout",
		Status:  "False",
		Timeout: 100 * time.Millisecond,
		CheckFunction: func() (bool, error) {
			return false, nil
		},
	}

	err = baseImpl.WaitForCondition(ctx, timeoutCondition)
	if err == nil {
		t.Fatal("Expected error from timeout")
	}

	// Test condition with error
	errorCondition := TestCondition{
		Type:    "Error",
		Status:  "Unknown",
		Timeout: 5 * time.Second,
		CheckFunction: func() (bool, error) {
			return false, fmt.Errorf("intentional error")
		},
	}

	err = baseImpl.WaitForCondition(ctx, errorCondition)
	if err == nil {
		t.Fatal("Expected error from check function")
	}
}

// TestBaseTestImplementationGetTestResults tests getting test results
func TestBaseTestImplementationGetTestResults(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	// Add some test data
	baseImpl.TestResults.AddLog("test log 1")
	baseImpl.TestResults.SetMetric("test_metric", 42)
	baseImpl.TestResults.IncrementResourceCount("node")

	results := baseImpl.GetTestResults()
	if results == nil {
		t.Fatal("Expected test results to be returned")
	}

	if len(results.Logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(results.Logs))
	}

	if results.Logs[0] != "test log 1" {
		t.Errorf("Expected log 'test log 1', got '%s'", results.Logs[0])
	}

	if results.Metrics["test_metric"] != 42 {
		t.Errorf("Expected metric 42, got %v", results.Metrics["test_metric"])
	}

	if results.ResourceCounts["node"] != 1 {
		t.Errorf("Expected node count 1, got %d", results.ResourceCounts["node"])
	}
}

// TestBaseTestImplementationResetTestState tests resetting test state
func TestBaseTestImplementationResetTestState(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	// Add some test data
	baseImpl.CreatedResources["node"] = []string{"test-node"}
	baseImpl.TestResults.AddLog("test log")
	baseImpl.TestResults.SetMetric("test_metric", 42)
	baseImpl.TestResults.IncrementResourceCount("node")

	err := baseImpl.ResetTestState()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify state is reset
	if len(baseImpl.CreatedResources) != 0 {
		t.Error("Expected created resources to be cleared")
	}

	if len(baseImpl.TestResults.Logs) != 1 {
		t.Errorf("Expected 1 log (reset message), got %d", len(baseImpl.TestResults.Logs))
	}

	if baseImpl.TestResults.Logs[0] != "Test state reset completed" {
		t.Errorf("Expected reset log message, got '%s'", baseImpl.TestResults.Logs[0])
	}

	if len(baseImpl.TestResults.Metrics) != 0 {
		t.Error("Expected metrics to be cleared")
	}

	if len(baseImpl.TestResults.ResourceCounts) != 0 {
		t.Error("Expected resource counts to be cleared")
	}
}

// TestNewFakeTestImplementation tests creating a new fake test implementation
func TestNewFakeTestImplementation(t *testing.T) {
	fakeImpl := NewFakeTestImplementation()

	if fakeImpl.BaseTestImplementation == nil {
		t.Error("Expected base test implementation to be set")
	}

	if fakeImpl.FakeCloud == nil {
		t.Error("Expected fake cloud to be set")
	}

	if fakeImpl.CloudProvider != fakeImpl.FakeCloud {
		t.Error("Expected cloud provider to be the fake cloud")
	}
}

// TestFakeTestImplementationGetFakeCloud tests getting the fake cloud
func TestFakeTestImplementationGetFakeCloud(t *testing.T) {
	fakeImpl := NewFakeTestImplementation()

	fakeCloud := fakeImpl.GetFakeCloud()
	if fakeCloud != fakeImpl.FakeCloud {
		t.Error("Expected fake cloud to be returned")
	}
}

// TestMockClientBuilder tests the mock client builder
func TestMockClientBuilder(t *testing.T) {
	mockBuilder := &MockClientBuilder{}

	// Test Config with default behavior
	config, err := mockBuilder.Config("test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config == nil {
		t.Error("Expected config to be returned")
	}

	// Test ConfigOrDie with default behavior
	configOrDie := mockBuilder.ConfigOrDie("test")
	if configOrDie == nil {
		t.Error("Expected config to be returned")
	}

	// Test Client with default behavior
	client, err := mockBuilder.Client("test")
	if err == nil {
		t.Fatal("Expected error from unconfigured client")
	}

	if client != nil {
		t.Error("Expected client to be nil")
	}

	// Test ClientOrDie with default behavior
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic from ClientOrDie")
		}
	}()
	mockBuilder.ClientOrDie("test")
}

// TestMockClientBuilderWithCustomFunctions tests the mock client builder with custom functions
func TestMockClientBuilderWithCustomFunctions(t *testing.T) {
	expectedConfig := &rest.Config{Host: "test-host"}
	expectedClient := fake.NewSimpleClientset()

	mockBuilder := &MockClientBuilder{
		ConfigFunc: func(name string) (*rest.Config, error) {
			return expectedConfig, nil
		},
		ConfigOrDieFunc: func(name string) *rest.Config {
			return expectedConfig
		},
		ClientFunc: func(name string) (kubernetes.Interface, error) {
			return expectedClient, nil
		},
		ClientOrDieFunc: func(name string) kubernetes.Interface {
			return expectedClient
		},
	}

	// Test Config with custom function
	config, err := mockBuilder.Config("test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config != expectedConfig {
		t.Error("Expected custom config to be returned")
	}

	// Test ConfigOrDie with custom function
	configOrDie := mockBuilder.ConfigOrDie("test")
	if configOrDie != expectedConfig {
		t.Error("Expected custom config to be returned")
	}

	// Test Client with custom function
	client, err := mockBuilder.Client("test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client != expectedClient {
		t.Error("Expected custom client to be returned")
	}

	// Test ClientOrDie with custom function
	clientOrDie := mockBuilder.ClientOrDie("test")
	if clientOrDie != expectedClient {
		t.Error("Expected custom client to be returned")
	}
}

// TestMockClientBuilderWithError tests the mock client builder with error
func TestMockClientBuilderWithError(t *testing.T) {
	expectedError := fmt.Errorf("intentional error")

	mockBuilder := &MockClientBuilder{
		ConfigFunc: func(name string) (*rest.Config, error) {
			return nil, expectedError
		},
		ClientFunc: func(name string) (kubernetes.Interface, error) {
			return nil, expectedError
		},
	}

	// Test Config with error
	config, err := mockBuilder.Config("test")
	if err != expectedError {
		t.Errorf("Expected error '%v', got '%v'", expectedError, err)
	}

	if config != nil {
		t.Error("Expected config to be nil")
	}

	// Test Client with error
	client, err := mockBuilder.Client("test")
	if err != expectedError {
		t.Errorf("Expected error '%v', got '%v'", expectedError, err)
	}

	if client != nil {
		t.Error("Expected client to be nil")
	}
}

// TestBaseTestImplementationConcurrency tests concurrent access to base test implementation
func TestBaseTestImplementationConcurrency(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	// Test concurrent access to test results
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			baseImpl.GetTestResults().AddLog(fmt.Sprintf("log %d", id))
			baseImpl.GetTestResults().SetMetric(fmt.Sprintf("metric_%d", id), id)
			baseImpl.GetTestResults().IncrementResourceCount("node")
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	results := baseImpl.GetTestResults()
	if len(results.Logs) != 10 {
		t.Errorf("Expected 10 logs, got %d", len(results.Logs))
	}

	if len(results.Metrics) != 10 {
		t.Errorf("Expected 10 metrics, got %d", len(results.Metrics))
	}

	if results.ResourceCounts["node"] != 10 {
		t.Errorf("Expected node count 10, got %d", results.ResourceCounts["node"])
	}
}

// TestBaseTestImplementationResourceTracking tests resource tracking functionality
func TestBaseTestImplementationResourceTracking(t *testing.T) {
	fakeCloud := &fakecloud.Cloud{}
	baseImpl := NewBaseTestImplementation(fakeCloud)

	ctx := context.Background()

	// Create multiple resources
	nodeConfig1 := &TestNodeConfig{Name: "node-1", ProviderID: "test://node-1"}
	nodeConfig2 := &TestNodeConfig{Name: "node-2", ProviderID: "test://node-2"}
	serviceConfig1 := &TestServiceConfig{Name: "service-1", Namespace: "default", Type: v1.ServiceTypeClusterIP}
	routeConfig1 := &TestRouteConfig{Name: "route-1", ClusterName: "test-cluster", TargetNode: "node-1", DestinationCIDR: "10.0.0.0/24"}

	_, err := baseImpl.CreateTestNode(ctx, nodeConfig1)
	if err != nil {
		t.Fatalf("Failed to create node 1: %v", err)
	}

	_, err = baseImpl.CreateTestNode(ctx, nodeConfig2)
	if err != nil {
		t.Fatalf("Failed to create node 2: %v", err)
	}

	_, err = baseImpl.CreateTestService(ctx, serviceConfig1)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	_, err = baseImpl.CreateTestRoute(ctx, routeConfig1)
	if err != nil {
		t.Fatalf("Failed to create route: %v", err)
	}

	// Verify resource tracking
	if len(baseImpl.CreatedResources["node"]) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(baseImpl.CreatedResources["node"]))
	}

	if len(baseImpl.CreatedResources["service"]) != 1 {
		t.Errorf("Expected 1 service, got %d", len(baseImpl.CreatedResources["service"]))
	}

	if len(baseImpl.CreatedResources["route"]) != 1 {
		t.Errorf("Expected 1 route, got %d", len(baseImpl.CreatedResources["route"]))
	}

	// Verify resource counts
	if baseImpl.TestResults.ResourceCounts["node"] != 2 {
		t.Errorf("Expected node count 2, got %d", baseImpl.TestResults.ResourceCounts["node"])
	}

	if baseImpl.TestResults.ResourceCounts["service"] != 1 {
		t.Errorf("Expected service count 1, got %d", baseImpl.TestResults.ResourceCounts["service"])
	}

	if baseImpl.TestResults.ResourceCounts["route"] != 1 {
		t.Errorf("Expected route count 1, got %d", baseImpl.TestResults.ResourceCounts["route"])
	}

	// Delete some resources
	err = baseImpl.DeleteTestNode(ctx, "node-1")
	if err != nil {
		t.Fatalf("Failed to delete node 1: %v", err)
	}

	err = baseImpl.DeleteTestService(ctx, "service-1")
	if err != nil {
		t.Fatalf("Failed to delete service: %v", err)
	}

	// Verify resources are removed from tracking
	if len(baseImpl.CreatedResources["node"]) != 1 {
		t.Errorf("Expected 1 remaining node, got %d", len(baseImpl.CreatedResources["node"]))
	}

	if baseImpl.CreatedResources["node"][0] != "node-2" {
		t.Errorf("Expected remaining node 'node-2', got '%s'", baseImpl.CreatedResources["node"][0])
	}

	if len(baseImpl.CreatedResources["service"]) != 0 {
		t.Errorf("Expected 0 remaining services, got %d", len(baseImpl.CreatedResources["service"]))
	}
}

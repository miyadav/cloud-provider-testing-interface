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
	"net"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	fakecloud "k8s.io/cloud-provider/fake"
)

// CreateExampleTestSuite demonstrates how to create a test suite for cloud provider testing.
func CreateExampleTestSuite() TestSuite {
	return TestSuite{
		Name:        "Basic Cloud Provider Tests",
		Description: "Tests basic cloud provider functionality",
		Setup: func(ti TestInterface) error {
			// Setup test environment
			config := &TestConfig{
				ProviderName:         "example-provider",
				ClusterName:          "test-cluster",
				Region:               "us-west-1",
				Zone:                 "us-west-1a",
				TestTimeout:          5 * time.Minute,
				CleanupResources:     true,
				MockExternalServices: true,
			}
			return ti.SetupTestEnvironment(config)
		},
		Teardown: func(ti TestInterface) error {
			// Clean up test environment
			return ti.TeardownTestEnvironment()
		},
		Tests: []Test{
			{
				Name:        "Test Load Balancer Creation",
				Description: "Tests that a load balancer can be created successfully",
				Run:         testLoadBalancerCreation,
				Timeout:     2 * time.Minute,
			},
			{
				Name:        "Test Node Registration",
				Description: "Tests that nodes can be registered with the cloud provider",
				Run:         testNodeRegistration,
				Timeout:     1 * time.Minute,
			},
			{
				Name:        "Test Route Management",
				Description: "Tests that routes can be created and managed",
				Run:         testRouteManagement,
				Timeout:     1 * time.Minute,
			},
		},
	}
}

// testLoadBalancerCreation tests load balancer creation functionality.
func testLoadBalancerCreation(ti TestInterface) error {
	ctx := context.Background()

	// Create a test node
	nodeConfig := &TestNodeConfig{
		Name:         "test-node-1",
		ProviderID:   "example://test-node-1",
		InstanceType: "t3.medium",
		Zone:         "us-west-1a",
		Region:       "us-west-1",
		Addresses: []v1.NodeAddress{
			{Type: v1.NodeInternalIP, Address: "10.0.0.1"},
			{Type: v1.NodeExternalIP, Address: "192.168.1.1"},
		},
	}

	node, err := ti.CreateTestNode(ctx, nodeConfig)
	if err != nil {
		return fmt.Errorf("failed to create test node: %w", err)
	}

	// Create a test service
	serviceConfig := &TestServiceConfig{
		Name:      "test-service",
		Namespace: "default",
		Type:      v1.ServiceTypeLoadBalancer,
		Ports: []v1.ServicePort{
			{Port: 80, TargetPort: intstr.FromInt(8080), Protocol: v1.ProtocolTCP},
		},
		ExternalTrafficPolicy: v1.ServiceExternalTrafficPolicyCluster,
	}

	service, err := ti.CreateTestService(ctx, serviceConfig)
	if err != nil {
		return fmt.Errorf("failed to create test service: %w", err)
	}

	// Test load balancer functionality
	cloud := ti.GetCloudProvider()
	loadBalancer, supported := cloud.LoadBalancer()
	if !supported {
		return fmt.Errorf("load balancer not supported by cloud provider")
	}

	// Test GetLoadBalancer
	status, exists, err := loadBalancer.GetLoadBalancer(ctx, "test-cluster", service)
	if err != nil {
		return fmt.Errorf("failed to get load balancer: %w", err)
	}
	_ = exists // Suppress unused variable warning

	// Test EnsureLoadBalancer
	nodes := []*v1.Node{node}
	status, err = loadBalancer.EnsureLoadBalancer(ctx, "test-cluster", service, nodes)
	if err != nil {
		return fmt.Errorf("failed to ensure load balancer: %w", err)
	}

	// Verify load balancer status
	if status == nil || len(status.Ingress) == 0 {
		return fmt.Errorf("load balancer status is empty")
	}

	// Test UpdateLoadBalancer
	err = loadBalancer.UpdateLoadBalancer(ctx, "test-cluster", service, nodes)
	if err != nil {
		return fmt.Errorf("failed to update load balancer: %w", err)
	}

	// Test EnsureLoadBalancerDeleted
	err = loadBalancer.EnsureLoadBalancerDeleted(ctx, "test-cluster", service)
	if err != nil {
		return fmt.Errorf("failed to delete load balancer: %w", err)
	}

	ti.GetTestResults().AddLog("Load balancer creation test completed successfully")
	return nil
}

// testNodeRegistration tests node registration functionality.
func testNodeRegistration(ti TestInterface) error {
	ctx := context.Background()

	// Create a test node
	nodeConfig := &TestNodeConfig{
		Name:         "test-node-2",
		ProviderID:   "example://test-node-2",
		InstanceType: "t3.large",
		Zone:         "us-west-1b",
		Region:       "us-west-1",
		Addresses: []v1.NodeAddress{
			{Type: v1.NodeInternalIP, Address: "10.0.0.2"},
			{Type: v1.NodeExternalIP, Address: "192.168.1.2"},
		},
		Labels: map[string]string{
			"node-role.kubernetes.io/worker": "true",
		},
	}

	node, err := ti.CreateTestNode(ctx, nodeConfig)
	if err != nil {
		return fmt.Errorf("failed to create test node: %w", err)
	}

	// Test instances functionality
	cloud := ti.GetCloudProvider()
	instances, supported := cloud.Instances()
	if !supported {
		return fmt.Errorf("instances not supported by cloud provider")
	}

	// Test NodeAddresses
	addresses, err := instances.NodeAddresses(ctx, types.NodeName(node.Name))
	if err != nil {
		return fmt.Errorf("failed to get node addresses: %w", err)
	}

	if len(addresses) == 0 {
		return fmt.Errorf("no addresses returned for node")
	}

	// Test NodeAddressesByProviderID
	addresses, err = instances.NodeAddressesByProviderID(ctx, node.Spec.ProviderID)
	if err != nil {
		return fmt.Errorf("failed to get node addresses by provider ID: %w", err)
	}

	// Test InstanceID
	instanceID, err := instances.InstanceID(ctx, types.NodeName(node.Name))
	if err != nil {
		return fmt.Errorf("failed to get instance ID: %w", err)
	}

	if instanceID == "" {
		return fmt.Errorf("empty instance ID returned")
	}

	// Test InstanceType
	instanceType, err := instances.InstanceType(ctx, types.NodeName(node.Name))
	if err != nil {
		return fmt.Errorf("failed to get instance type: %w", err)
	}

	if instanceType == "" {
		return fmt.Errorf("empty instance type returned")
	}

	// Test InstanceExistsByProviderID
	exists, err := instances.InstanceExistsByProviderID(ctx, node.Spec.ProviderID)
	if err != nil {
		return fmt.Errorf("failed to check instance existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("instance should exist but doesn't")
	}

	ti.GetTestResults().AddLog("Node registration test completed successfully")
	return nil
}

// testRouteManagement tests route management functionality.
func testRouteManagement(ti TestInterface) error {
	ctx := context.Background()

	// Create a test node
	nodeConfig := &TestNodeConfig{
		Name:         "test-node-3",
		ProviderID:   "example://test-node-3",
		InstanceType: "t3.medium",
		Zone:         "us-west-1a",
		Region:       "us-west-1",
	}

	node, err := ti.CreateTestNode(ctx, nodeConfig)
	if err != nil {
		return fmt.Errorf("failed to create test node: %w", err)
	}

	// Test routes functionality
	cloud := ti.GetCloudProvider()
	routes, supported := cloud.Routes()
	if !supported {
		return fmt.Errorf("routes not supported by cloud provider")
	}

	// Test ListRoutes
	existingRoutes, err := routes.ListRoutes(ctx, "test-cluster")
	if err != nil {
		return fmt.Errorf("failed to list routes: %w", err)
	}

	// Create a test route
	routeConfig := &TestRouteConfig{
		Name:            "test-route",
		ClusterName:     "test-cluster",
		TargetNode:      types.NodeName(node.Name),
		DestinationCIDR: "10.100.0.0/24",
		Blackhole:       false,
	}

	route, err := ti.CreateTestRoute(ctx, routeConfig)
	if err != nil {
		return fmt.Errorf("failed to create test route: %w", err)
	}

	// Test CreateRoute
	err = routes.CreateRoute(ctx, "test-cluster", "test-route", route)
	if err != nil {
		return fmt.Errorf("failed to create route: %w", err)
	}

	// Test ListRoutes again to verify route was created
	updatedRoutes, err := routes.ListRoutes(ctx, "test-cluster")
	if err != nil {
		return fmt.Errorf("failed to list routes after creation: %w", err)
	}

	if len(updatedRoutes) <= len(existingRoutes) {
		return fmt.Errorf("route was not created")
	}

	// Test DeleteRoute
	err = routes.DeleteRoute(ctx, "test-cluster", route)
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}

	ti.GetTestResults().AddLog("Route management test completed successfully")
	return nil
}

// ExampleTestRunner demonstrates how to use the TestRunner to run tests.
func ExampleTestRunner() {
	// Create a fake test implementation
	fakeImpl := NewFakeTestImplementation()

	// Create a test runner
	runner := NewTestRunner(fakeImpl)

	// Add test suites
	runner.AddTestSuite(CreateExampleTestSuite())

	// Run tests
	ctx := context.Background()
	err := runner.RunTests(ctx)
	if err != nil {
		fmt.Printf("Test execution failed: %v\n", err)
		return
	}

	// Get results
	results := runner.GetResults()
	summary := runner.GetSummary()

	fmt.Printf("Test Summary:\n")
	fmt.Printf("  Total Tests: %d\n", summary.TotalTests)
	fmt.Printf("  Passed: %d\n", summary.PassedTests)
	fmt.Printf("  Failed: %d\n", summary.FailedTests)
	fmt.Printf("  Skipped: %d\n", summary.SkippedTests)
	fmt.Printf("  Duration: %v\n", summary.TotalDuration)

	// Print detailed results
	for _, result := range results {
		status := "PASSED"
		if !result.Success {
			status = "FAILED"
		}
		if result.Test.Skip {
			status = "SKIPPED"
		}
		fmt.Printf("  %s: %s (%v)\n", status, result.Test.Name, result.Duration)
	}
}

// TestExampleTestSuite runs the example test suite to verify it works.
func TestExampleTestSuite(t *testing.T) {
	// Create a fake test implementation
	fakeImpl := NewFakeTestImplementation()

	// Configure the fake cloud provider for testing
	fakeCloud := fakeImpl.GetFakeCloud()
	fakeCloud.SetNodeAddresses([]v1.NodeAddress{
		{Type: v1.NodeInternalIP, Address: "10.0.0.1"},
		{Type: v1.NodeExternalIP, Address: "192.168.1.1"},
	})
	fakeCloud.ExtID[types.NodeName("test-node-1")] = "test-node-1"
	fakeCloud.ExtID[types.NodeName("test-node-2")] = "test-node-2"
	fakeCloud.ExtID[types.NodeName("test-node-3")] = "test-node-3"
	fakeCloud.InstanceTypes[types.NodeName("test-node-1")] = "t3.medium"
	fakeCloud.InstanceTypes[types.NodeName("test-node-2")] = "t3.large"
	fakeCloud.InstanceTypes[types.NodeName("test-node-3")] = "t3.medium"
	fakeCloud.ExistsByProviderID = true
	fakeCloud.ExternalIP = net.ParseIP("192.168.1.100")

	// Create a test runner
	runner := NewTestRunner(fakeImpl)

	// Add test suites
	runner.AddTestSuite(CreateExampleTestSuite())

	// Run tests
	ctx := context.Background()
	err := runner.RunTests(ctx)
	if err != nil {
		t.Fatalf("Test execution failed: %v", err)
	}

	// Verify results
	summary := runner.GetSummary()

	if summary.TotalTests == 0 {
		t.Error("No tests were executed")
	}

	if summary.FailedTests > 0 {
		t.Errorf("Some tests failed: %d failed out of %d total", summary.FailedTests, summary.TotalTests)
	}

	// Print results for debugging
	t.Logf("Test Summary: %d total, %d passed, %d failed, %d skipped",
		summary.TotalTests, summary.PassedTests, summary.FailedTests, summary.SkippedTests)
}

// TestBaseTestImplementation tests the base test implementation.
func TestBaseTestImplementation(t *testing.T) {
	// Create a fake cloud provider
	fakeCloud := &fakecloud.Cloud{
		Balancers:     make(map[string]fakecloud.Balancer),
		ExtID:         make(map[types.NodeName]string),
		ExtIDErr:      make(map[types.NodeName]error),
		InstanceTypes: make(map[types.NodeName]string),
		ProviderID:    make(map[types.NodeName]string),
		RouteMap:      make(map[string]*fakecloud.Route),
	}

	// Create base test implementation
	baseImpl := NewBaseTestImplementation(fakeCloud)

	// Test setup
	config := &TestConfig{
		ProviderName:         "test-provider",
		ClusterName:          "test-cluster",
		Region:               "us-west-1",
		Zone:                 "us-west-1a",
		TestTimeout:          1 * time.Minute,
		CleanupResources:     true,
		MockExternalServices: true,
		ClientBuilder:        &MockClientBuilder{},
		InformerFactory:      informers.NewSharedInformerFactory(fake.NewSimpleClientset(), 0),
	}

	err := baseImpl.SetupTestEnvironment(config)
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}

	// Test node creation
	nodeConfig := &TestNodeConfig{
		Name:         "test-node",
		ProviderID:   "test://test-node",
		InstanceType: "t3.medium",
		Zone:         "us-west-1a",
		Region:       "us-west-1",
	}

	node, err := baseImpl.CreateTestNode(context.Background(), nodeConfig)
	if err != nil {
		t.Fatalf("Failed to create test node: %v", err)
	}

	if node.Name != "test-node" {
		t.Errorf("Expected node name 'test-node', got '%s'", node.Name)
	}

	// Test service creation
	serviceConfig := &TestServiceConfig{
		Name:      "test-service",
		Namespace: "default",
		Type:      v1.ServiceTypeLoadBalancer,
		Ports: []v1.ServicePort{
			{Port: 80, TargetPort: intstr.FromInt(8080), Protocol: v1.ProtocolTCP},
		},
	}

	service, err := baseImpl.CreateTestService(context.Background(), serviceConfig)
	if err != nil {
		t.Fatalf("Failed to create test service: %v", err)
	}

	if service.Name != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", service.Name)
	}

	// Test teardown
	err = baseImpl.TeardownTestEnvironment()
	if err != nil {
		t.Fatalf("Failed to teardown test environment: %v", err)
	}

	// Test results
	results := baseImpl.GetTestResults()
	if results == nil {
		t.Error("Test results should not be nil")
	}

	if len(results.Logs) == 0 {
		t.Error("Test logs should not be empty")
	}
}

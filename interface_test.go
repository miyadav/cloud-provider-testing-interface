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
	"k8s.io/client-go/kubernetes/fake"
)

// TestTestResults tests the TestResults functionality
func TestTestResults(t *testing.T) {
	results := &TestResults{}

	// Test AddLog
	results.AddLog("test log 1")
	results.AddLog("test log 2")

	if len(results.Logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(results.Logs))
	}

	if results.Logs[0] != "test log 1" {
		t.Errorf("Expected first log to be 'test log 1', got '%s'", results.Logs[0])
	}

	// Test SetMetric
	results.SetMetric("test_metric", 42)
	results.SetMetric("string_metric", "test_value")

	if results.Metrics["test_metric"] != 42 {
		t.Errorf("Expected metric 'test_metric' to be 42, got %v", results.Metrics["test_metric"])
	}

	if results.Metrics["string_metric"] != "test_value" {
		t.Errorf("Expected metric 'string_metric' to be 'test_value', got %v", results.Metrics["string_metric"])
	}

	// Test IncrementResourceCount
	results.IncrementResourceCount("node")
	results.IncrementResourceCount("node")
	results.IncrementResourceCount("service")

	if results.ResourceCounts["node"] != 2 {
		t.Errorf("Expected node count to be 2, got %d", results.ResourceCounts["node"])
	}

	if results.ResourceCounts["service"] != 1 {
		t.Errorf("Expected service count to be 1, got %d", results.ResourceCounts["service"])
	}
}

// TestTestRunner tests the TestRunner functionality
func TestTestRunner(t *testing.T) {
	// Create a fake test implementation
	fakeImpl := NewFakeTestImplementation()

	// Create a test runner
	runner := NewTestRunner(fakeImpl)

	if runner.TestInterface == nil {
		t.Error("TestInterface should not be nil")
	}

	if len(runner.TestSuites) != 0 {
		t.Error("TestSuites should be empty initially")
	}

	if len(runner.Results) != 0 {
		t.Error("Results should be empty initially")
	}
}

// TestTestRunnerAddTestSuite tests adding test suites to the runner
func TestTestRunnerAddTestSuite(t *testing.T) {
	fakeImpl := NewFakeTestImplementation()
	runner := NewTestRunner(fakeImpl)

	suite := TestSuite{
		Name:        "Test Suite 1",
		Description: "Test suite description",
		Tests:       []Test{},
	}

	runner.AddTestSuite(suite)

	if len(runner.TestSuites) != 1 {
		t.Errorf("Expected 1 test suite, got %d", len(runner.TestSuites))
	}

	if runner.TestSuites[0].Name != "Test Suite 1" {
		t.Errorf("Expected test suite name 'Test Suite 1', got '%s'", runner.TestSuites[0].Name)
	}
}

// TestTestRunnerRunTests tests running tests with the runner
func TestTestRunnerRunTests(t *testing.T) {
	fakeImpl := NewFakeTestImplementation()
	runner := NewTestRunner(fakeImpl)

	// Create a simple test suite
	suite := TestSuite{
		Name:        "Simple Test Suite",
		Description: "A simple test suite",
		Setup: func(ti TestInterface) error {
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
			return ti.SetupTestEnvironment(config)
		},
		Teardown: func(ti TestInterface) error {
			return ti.TeardownTestEnvironment()
		},
		Tests: []Test{
			{
				Name:        "Simple Test",
				Description: "A simple test that always passes",
				Run: func(ti TestInterface) error {
					ti.GetTestResults().AddLog("Simple test executed successfully")
					return nil
				},
				Timeout: 30 * time.Second,
			},
		},
	}

	runner.AddTestSuite(suite)

	ctx := context.Background()
	err := runner.RunTests(ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	results := runner.GetResults()
	if len(results) != 1 {
		t.Errorf("Expected 1 test result, got %d", len(results))
	}

	if !results[0].Success {
		t.Error("Expected test to pass")
	}

	summary := runner.GetSummary()
	if summary.TotalTests != 1 {
		t.Errorf("Expected 1 total test, got %d", summary.TotalTests)
	}

	if summary.PassedTests != 1 {
		t.Errorf("Expected 1 passed test, got %d", summary.PassedTests)
	}
}

// TestTestRunnerRunTestsWithFailure tests running tests that fail
func TestTestRunnerRunTestsWithFailure(t *testing.T) {
	fakeImpl := NewFakeTestImplementation()
	runner := NewTestRunner(fakeImpl)

	suite := TestSuite{
		Name:        "Failure Test Suite",
		Description: "A test suite with a failing test",
		Setup: func(ti TestInterface) error {
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
			return ti.SetupTestEnvironment(config)
		},
		Teardown: func(ti TestInterface) error {
			return ti.TeardownTestEnvironment()
		},
		Tests: []Test{
			{
				Name:        "Failing Test",
				Description: "A test that always fails",
				Run: func(ti TestInterface) error {
					return fmt.Errorf("intentional test failure")
				},
				Timeout: 30 * time.Second,
			},
		},
	}

	runner.AddTestSuite(suite)

	ctx := context.Background()
	err := runner.RunTests(ctx)

	if err == nil {
		t.Fatal("Expected error from failing test")
	}

	results := runner.GetResults()
	if len(results) != 1 {
		t.Errorf("Expected 1 test result, got %d", len(results))
	}

	if results[0].Success {
		t.Error("Expected test to fail")
	}

	summary := runner.GetSummary()
	if summary.FailedTests != 1 {
		t.Errorf("Expected 1 failed test, got %d", summary.FailedTests)
	}
}

// TestTestRunnerRunTestsWithSkipped tests running tests that are skipped
func TestTestRunnerRunTestsWithSkipped(t *testing.T) {
	fakeImpl := NewFakeTestImplementation()
	runner := NewTestRunner(fakeImpl)

	suite := TestSuite{
		Name:        "Skipped Test Suite",
		Description: "A test suite with a skipped test",
		Setup: func(ti TestInterface) error {
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
			return ti.SetupTestEnvironment(config)
		},
		Teardown: func(ti TestInterface) error {
			return ti.TeardownTestEnvironment()
		},
		Tests: []Test{
			{
				Name:        "Skipped Test",
				Description: "A test that is skipped",
				Skip:        true,
				SkipReason:  "Test is not implemented yet",
				Run: func(ti TestInterface) error {
					return fmt.Errorf("this should not be executed")
				},
				Timeout: 30 * time.Second,
			},
		},
	}

	runner.AddTestSuite(suite)

	ctx := context.Background()
	err := runner.RunTests(ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	results := runner.GetResults()
	if len(results) != 1 {
		t.Errorf("Expected 1 test result, got %d", len(results))
	}

	if !results[0].Success {
		t.Error("Expected skipped test to be marked as successful")
	}

	summary := runner.GetSummary()
	if summary.SkippedTests != 1 {
		t.Errorf("Expected 1 skipped test, got %d", summary.SkippedTests)
	}
}

// TestTestRunnerRunTestsWithTimeout tests running tests with timeout
func TestTestRunnerRunTestsWithTimeout(t *testing.T) {
	fakeImpl := NewFakeTestImplementation()
	runner := NewTestRunner(fakeImpl)

	suite := TestSuite{
		Name:        "Timeout Test Suite",
		Description: "A test suite with a timeout test",
		Setup: func(ti TestInterface) error {
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
			return ti.SetupTestEnvironment(config)
		},
		Teardown: func(ti TestInterface) error {
			return ti.TeardownTestEnvironment()
		},
		Tests: []Test{
			{
				Name:        "Timeout Test",
				Description: "A test that times out",
				Run: func(ti TestInterface) error {
					// Create a context with timeout for the test function
					ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
					defer cancel()

					// Sleep and check for context cancellation
					select {
					case <-time.After(200 * time.Millisecond):
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				},
				Timeout: 100 * time.Millisecond, // Very short timeout
			},
		},
	}

	runner.AddTestSuite(suite)

	ctx := context.Background()
	err := runner.RunTests(ctx)

	if err == nil {
		t.Fatal("Expected error from timeout")
	}

	results := runner.GetResults()
	if len(results) != 1 {
		t.Errorf("Expected 1 test result, got %d", len(results))
	}

	if results[0].Success {
		t.Error("Expected test to fail due to timeout")
	}
}

// TestTestRunnerRunTestsWithCleanup tests running tests with cleanup functions
func TestTestRunnerRunTestsWithCleanup(t *testing.T) {
	fakeImpl := NewFakeTestImplementation()
	runner := NewTestRunner(fakeImpl)

	cleanupCalled := false

	suite := TestSuite{
		Name:        "Cleanup Test Suite",
		Description: "A test suite with cleanup functions",
		Setup: func(ti TestInterface) error {
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
			return ti.SetupTestEnvironment(config)
		},
		Teardown: func(ti TestInterface) error {
			return ti.TeardownTestEnvironment()
		},
		Tests: []Test{
			{
				Name:        "Cleanup Test",
				Description: "A test with cleanup function",
				Run: func(ti TestInterface) error {
					ti.GetTestResults().AddLog("Test executed")
					return nil
				},
				Cleanup: func(ti TestInterface) error {
					cleanupCalled = true
					ti.GetTestResults().AddLog("Cleanup executed")
					return nil
				},
				Timeout: 30 * time.Second,
			},
		},
	}

	runner.AddTestSuite(suite)

	ctx := context.Background()
	err := runner.RunTests(ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !cleanupCalled {
		t.Error("Expected cleanup function to be called")
	}

	results := runner.GetResults()
	if len(results) != 1 {
		t.Errorf("Expected 1 test result, got %d", len(results))
	}

	if !results[0].Success {
		t.Error("Expected test to pass")
	}
}

// TestTestRunnerRunTestsWithSuiteSetupTeardown tests running tests with suite setup and teardown
func TestTestRunnerRunTestsWithSuiteSetupTeardown(t *testing.T) {
	fakeImpl := NewFakeTestImplementation()
	runner := NewTestRunner(fakeImpl)

	setupCalled := false
	teardownCalled := false

	suite := TestSuite{
		Name:        "Suite Setup Teardown Test",
		Description: "A test suite with setup and teardown",
		Setup: func(ti TestInterface) error {
			setupCalled = true
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
			return ti.SetupTestEnvironment(config)
		},
		Teardown: func(ti TestInterface) error {
			teardownCalled = true
			return ti.TeardownTestEnvironment()
		},
		Tests: []Test{
			{
				Name:        "Suite Test",
				Description: "A test in a suite with setup/teardown",
				Run: func(ti TestInterface) error {
					ti.GetTestResults().AddLog("Suite test executed")
					return nil
				},
				Timeout: 30 * time.Second,
			},
		},
	}

	runner.AddTestSuite(suite)

	ctx := context.Background()
	err := runner.RunTests(ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !setupCalled {
		t.Error("Expected suite setup to be called")
	}

	if !teardownCalled {
		t.Error("Expected suite teardown to be called")
	}

	results := runner.GetResults()
	if len(results) != 1 {
		t.Errorf("Expected 1 test result, got %d", len(results))
	}

	if !results[0].Success {
		t.Error("Expected test to pass")
	}
}

// TestTestRunnerRunTestsWithSuiteSetupFailure tests running tests with suite setup failure
func TestTestRunnerRunTestsWithSuiteSetupFailure(t *testing.T) {
	fakeImpl := NewFakeTestImplementation()
	runner := NewTestRunner(fakeImpl)

	suite := TestSuite{
		Name:        "Suite Setup Failure Test",
		Description: "A test suite with setup failure",
		Setup: func(ti TestInterface) error {
			return fmt.Errorf("intentional setup failure")
		},
		Tests: []Test{
			{
				Name:        "Should Not Run",
				Description: "This test should not run due to setup failure",
				Run: func(ti TestInterface) error {
					return fmt.Errorf("this should not be executed")
				},
				Timeout: 30 * time.Second,
			},
		},
	}

	runner.AddTestSuite(suite)

	ctx := context.Background()
	err := runner.RunTests(ctx)

	if err == nil {
		t.Fatal("Expected error from setup failure")
	}

	results := runner.GetResults()
	if len(results) != 0 {
		t.Errorf("Expected 0 test results due to setup failure, got %d", len(results))
	}
}

// TestTestRunnerRunTestsWithSuiteTeardownFailure tests running tests with suite teardown failure
func TestTestRunnerRunTestsWithSuiteTeardownFailure(t *testing.T) {
	fakeImpl := NewFakeTestImplementation()
	runner := NewTestRunner(fakeImpl)

	suite := TestSuite{
		Name:        "Suite Teardown Failure Test",
		Description: "A test suite with teardown failure",
		Setup: func(ti TestInterface) error {
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
			return ti.SetupTestEnvironment(config)
		},
		Teardown: func(ti TestInterface) error {
			return fmt.Errorf("intentional teardown failure")
		},
		Tests: []Test{
			{
				Name:        "Suite Test",
				Description: "A test in a suite with teardown failure",
				Run: func(ti TestInterface) error {
					ti.GetTestResults().AddLog("Suite test executed")
					return nil
				},
				Timeout: 30 * time.Second,
			},
		},
	}

	runner.AddTestSuite(suite)

	ctx := context.Background()
	err := runner.RunTests(ctx)

	if err == nil {
		t.Fatal("Expected error from teardown failure")
	}

	results := runner.GetResults()
	if len(results) != 1 {
		t.Errorf("Expected 1 test result, got %d", len(results))
	}

	if !results[0].Success {
		t.Error("Expected test to pass despite teardown failure")
	}
}

// TestTestRunnerGetSummary tests getting test summary
func TestTestRunnerGetSummary(t *testing.T) {
	fakeImpl := NewFakeTestImplementation()
	runner := NewTestRunner(fakeImpl)

	// Manually add some results
	runner.Results = []TestResult{
		{
			Test:     Test{Name: "Passed Test 1"},
			Success:  true,
			Duration: 1 * time.Second,
		},
		{
			Test:     Test{Name: "Passed Test 2"},
			Success:  true,
			Duration: 2 * time.Second,
		},
		{
			Test:     Test{Name: "Failed Test"},
			Success:  false,
			Duration: 1 * time.Second,
		},
		{
			Test:     Test{Name: "Skipped Test", Skip: true},
			Success:  true,
			Duration: 0,
		},
	}

	summary := runner.GetSummary()

	if summary.TotalTests != 4 {
		t.Errorf("Expected 4 total tests, got %d", summary.TotalTests)
	}

	if summary.PassedTests != 2 {
		t.Errorf("Expected 2 passed tests, got %d", summary.PassedTests)
	}

	if summary.FailedTests != 1 {
		t.Errorf("Expected 1 failed test, got %d", summary.FailedTests)
	}

	if summary.SkippedTests != 1 {
		t.Errorf("Expected 1 skipped test, got %d", summary.SkippedTests)
	}

	expectedDuration := 4 * time.Second
	if summary.TotalDuration != expectedDuration {
		t.Errorf("Expected total duration %v, got %v", expectedDuration, summary.TotalDuration)
	}
}

// TestTestConfigValidation tests TestConfig validation
func TestTestConfigValidation(t *testing.T) {
	config := &TestConfig{
		ProviderName:         "test-provider",
		ClusterName:          "test-cluster",
		Region:               "us-west-1",
		Zone:                 "us-west-1a",
		TestTimeout:          5 * time.Minute,
		CleanupResources:     true,
		MockExternalServices: true,
		TestData: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	if config.ProviderName != "test-provider" {
		t.Errorf("Expected provider name 'test-provider', got '%s'", config.ProviderName)
	}

	if config.ClusterName != "test-cluster" {
		t.Errorf("Expected cluster name 'test-cluster', got '%s'", config.ClusterName)
	}

	if config.Region != "us-west-1" {
		t.Errorf("Expected region 'us-west-1', got '%s'", config.Region)
	}

	if config.Zone != "us-west-1a" {
		t.Errorf("Expected zone 'us-west-1a', got '%s'", config.Zone)
	}

	if config.TestTimeout != 5*time.Minute {
		t.Errorf("Expected timeout 5 minutes, got %v", config.TestTimeout)
	}

	if !config.CleanupResources {
		t.Error("Expected cleanup resources to be true")
	}

	if !config.MockExternalServices {
		t.Error("Expected mock external services to be true")
	}

	if len(config.TestData) != 2 {
		t.Errorf("Expected 2 test data items, got %d", len(config.TestData))
	}
}

// TestTestNodeConfigValidation tests TestNodeConfig validation
func TestTestNodeConfigValidation(t *testing.T) {
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

	if nodeConfig.Name != "test-node" {
		t.Errorf("Expected node name 'test-node', got '%s'", nodeConfig.Name)
	}

	if nodeConfig.ProviderID != "test://test-node" {
		t.Errorf("Expected provider ID 'test://test-node', got '%s'", nodeConfig.ProviderID)
	}

	if nodeConfig.InstanceType != "t3.medium" {
		t.Errorf("Expected instance type 't3.medium', got '%s'", nodeConfig.InstanceType)
	}

	if nodeConfig.Zone != "us-west-1a" {
		t.Errorf("Expected zone 'us-west-1a', got '%s'", nodeConfig.Zone)
	}

	if nodeConfig.Region != "us-west-1" {
		t.Errorf("Expected region 'us-west-1', got '%s'", nodeConfig.Region)
	}

	if len(nodeConfig.Addresses) != 2 {
		t.Errorf("Expected 2 addresses, got %d", len(nodeConfig.Addresses))
	}

	if len(nodeConfig.Labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(nodeConfig.Labels))
	}

	if len(nodeConfig.Annotations) != 1 {
		t.Errorf("Expected 1 annotation, got %d", len(nodeConfig.Annotations))
	}

	if len(nodeConfig.Conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(nodeConfig.Conditions))
	}
}

// TestTestServiceConfigValidation tests TestServiceConfig validation
func TestTestServiceConfigValidation(t *testing.T) {
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

	if serviceConfig.Name != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", serviceConfig.Name)
	}

	if serviceConfig.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", serviceConfig.Namespace)
	}

	if serviceConfig.Type != v1.ServiceTypeLoadBalancer {
		t.Errorf("Expected service type LoadBalancer, got %v", serviceConfig.Type)
	}

	if len(serviceConfig.Ports) != 2 {
		t.Errorf("Expected 2 ports, got %d", len(serviceConfig.Ports))
	}

	if serviceConfig.LoadBalancerIP != "192.168.1.100" {
		t.Errorf("Expected load balancer IP '192.168.1.100', got '%s'", serviceConfig.LoadBalancerIP)
	}

	if serviceConfig.ExternalTrafficPolicy != v1.ServiceExternalTrafficPolicyCluster {
		t.Errorf("Expected external traffic policy Cluster, got %v", serviceConfig.ExternalTrafficPolicy)
	}

	if *serviceConfig.InternalTrafficPolicy != v1.ServiceInternalTrafficPolicyCluster {
		t.Errorf("Expected internal traffic policy Cluster, got %v", *serviceConfig.InternalTrafficPolicy)
	}

	if len(serviceConfig.Labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(serviceConfig.Labels))
	}

	if len(serviceConfig.Annotations) != 1 {
		t.Errorf("Expected 1 annotation, got %d", len(serviceConfig.Annotations))
	}
}

// TestTestRouteConfigValidation tests TestRouteConfig validation
func TestTestRouteConfigValidation(t *testing.T) {
	routeConfig := &TestRouteConfig{
		Name:            "test-route",
		ClusterName:     "test-cluster",
		TargetNode:      types.NodeName("test-node"),
		DestinationCIDR: "10.100.0.0/24",
		Blackhole:       false,
	}

	if routeConfig.Name != "test-route" {
		t.Errorf("Expected route name 'test-route', got '%s'", routeConfig.Name)
	}

	if routeConfig.ClusterName != "test-cluster" {
		t.Errorf("Expected cluster name 'test-cluster', got '%s'", routeConfig.ClusterName)
	}

	if routeConfig.TargetNode != types.NodeName("test-node") {
		t.Errorf("Expected target node 'test-node', got '%s'", routeConfig.TargetNode)
	}

	if routeConfig.DestinationCIDR != "10.100.0.0/24" {
		t.Errorf("Expected destination CIDR '10.100.0.0/24', got '%s'", routeConfig.DestinationCIDR)
	}

	if routeConfig.Blackhole {
		t.Error("Expected blackhole to be false")
	}
}

// TestTestConditionValidation tests TestCondition validation
func TestTestConditionValidation(t *testing.T) {
	condition := &TestCondition{
		Type:    "Ready",
		Status:  "True",
		Reason:  "NodeReady",
		Message: "Node is ready",
		Timeout: 30 * time.Second,
		CheckFunction: func() (bool, error) {
			return true, nil
		},
	}

	if condition.Type != "Ready" {
		t.Errorf("Expected condition type 'Ready', got '%s'", condition.Type)
	}

	if condition.Status != "True" {
		t.Errorf("Expected condition status 'True', got '%s'", condition.Status)
	}

	if condition.Reason != "NodeReady" {
		t.Errorf("Expected condition reason 'NodeReady', got '%s'", condition.Reason)
	}

	if condition.Message != "Node is ready" {
		t.Errorf("Expected condition message 'Node is ready', got '%s'", condition.Message)
	}

	if condition.Timeout != 30*time.Second {
		t.Errorf("Expected condition timeout 30 seconds, got %v", condition.Timeout)
	}

	if condition.CheckFunction == nil {
		t.Error("Expected check function to be set")
	}

	met, err := condition.CheckFunction()
	if err != nil {
		t.Errorf("Expected no error from check function, got %v", err)
	}

	if !met {
		t.Error("Expected condition to be met")
	}
}

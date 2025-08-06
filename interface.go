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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	cloudprovider "k8s.io/cloud-provider"
)

// TestInterface is an abstract, pluggable interface for testing cloud providers.
// This interface provides a cloud-agnostic way to test cloud provider implementations
// by abstracting away the specific cloud provider details and focusing on the
// behavior and functionality that should be consistent across all cloud providers.
type TestInterface interface {
	// SetupTestEnvironment initializes the test environment with the given configuration.
	// This should create any necessary test resources, mock services, or test data.
	SetupTestEnvironment(config *TestConfig) error

	// TeardownTestEnvironment cleans up the test environment and removes any test resources.
	TeardownTestEnvironment() error

	// GetCloudProvider returns the cloud provider instance to be tested.
	// This allows the test framework to access the actual cloud provider implementation.
	GetCloudProvider() cloudprovider.Interface

	// CreateTestNode creates a test node with the specified configuration.
	// The node should be created in a way that simulates a real node in the cloud provider.
	CreateTestNode(ctx context.Context, nodeConfig *TestNodeConfig) (*v1.Node, error)

	// DeleteTestNode deletes a test node.
	DeleteTestNode(ctx context.Context, nodeName string) error

	// CreateTestService creates a test service with the specified configuration.
	// The service should be created in a way that simulates a real service in the cloud provider.
	CreateTestService(ctx context.Context, serviceConfig *TestServiceConfig) (*v1.Service, error)

	// DeleteTestService deletes a test service.
	DeleteTestService(ctx context.Context, serviceName string) error

	// CreateTestRoute creates a test route with the specified configuration.
	CreateTestRoute(ctx context.Context, routeConfig *TestRouteConfig) (*cloudprovider.Route, error)

	// DeleteTestRoute deletes a test route.
	DeleteTestRoute(ctx context.Context, routeName string) error

	WaitForCondition(ctx context.Context, condition TestCondition) error

	// GetTestResults returns the results of the test execution.
	GetTestResults() *TestResults

	// ResetTestState resets the test state to a clean state.
	ResetTestState() error
}

// TestConfig holds the configuration for a test environment.
type TestConfig struct {
	// ProviderName is the name of the cloud provider being tested.
	ProviderName string

	// ClusterName is the name of the test cluster.
	ClusterName string

	// Region is the region where the test resources should be created.
	Region string

	// Zone is the zone where the test resources should be created.
	Zone string

	// ClientBuilder is the client builder for creating Kubernetes clients.
	ClientBuilder cloudprovider.ControllerClientBuilder

	// InformerFactory is the informer factory for creating informers.
	InformerFactory informers.SharedInformerFactory

	// TestTimeout is the timeout for test operations.
	TestTimeout time.Duration

	// CleanupResources determines whether to clean up resources after tests.
	CleanupResources bool

	// MockExternalServices determines whether to use mock external services.
	MockExternalServices bool

	// TestData contains additional test-specific configuration.
	TestData map[string]interface{}
}

// TestNodeConfig holds the configuration for creating a test node.
type TestNodeConfig struct {
	// Name is the name of the test node.
	Name string

	// ProviderID is the provider ID of the node.
	ProviderID string

	// InstanceType is the instance type of the node.
	InstanceType string

	// Zone is the zone where the node should be created.
	Zone string

	// Region is the region where the node should be created.
	Region string

	// Addresses are the network addresses of the node.
	Addresses []v1.NodeAddress

	// Labels are the labels to be applied to the node.
	Labels map[string]string

	// Annotations are the annotations to be applied to the node.
	Annotations map[string]string

	// Conditions are the conditions of the node.
	Conditions []v1.NodeCondition
}

// TestServiceConfig holds the configuration for creating a test service.
type TestServiceConfig struct {
	// Name is the name of the test service.
	Name string

	// Namespace is the namespace of the service.
	Namespace string

	// Type is the type of the service.
	Type v1.ServiceType

	// Ports are the ports of the service.
	Ports []v1.ServicePort

	// LoadBalancerIP is the IP address for the load balancer.
	LoadBalancerIP string

	// ExternalTrafficPolicy is the external traffic policy.
	ExternalTrafficPolicy v1.ServiceExternalTrafficPolicy

	// InternalTrafficPolicy is the internal traffic policy.
	InternalTrafficPolicy *v1.ServiceInternalTrafficPolicy

	// Labels are the labels to be applied to the service.
	Labels map[string]string

	// Annotations are the annotations to be applied to the service.
	Annotations map[string]string
}

// TestRouteConfig holds the configuration for creating a test route.
type TestRouteConfig struct {
	// Name is the name of the test route.
	Name string

	// ClusterName is the name of the cluster.
	ClusterName string

	// TargetNode is the target node for the route.
	TargetNode types.NodeName

	// DestinationCIDR is the destination CIDR for the route.
	DestinationCIDR string

	// Blackhole determines whether this is a blackhole route.
	Blackhole bool
}

// TestCondition represents a condition that should be met during testing.
type TestCondition struct {
	// Type is the type of the condition.
	Type string

	// Status is the status of the condition.
	Status string

	// Reason is the reason for the condition.
	Reason string

	// Message is the message of the condition.
	Message string

	// Timeout is the timeout for the condition.
	Timeout time.Duration

	// CheckFunction is a custom function to check the condition.
	CheckFunction func() (bool, error)
}

// TestResults holds the results of a test execution.
type TestResults struct {
	// Success indicates whether the test was successful.
	Success bool

	// Error is the error that occurred during the test.
	Error error

	// Duration is the duration of the test.
	Duration time.Duration

	// ResourceCounts contains counts of resources created during the test.
	ResourceCounts map[string]int

	// Metrics contains test-specific metrics.
	Metrics map[string]interface{}

	// Logs contains test logs.
	Logs []string

	// mu protects access to the TestResults fields
	mu sync.RWMutex
}

// AddLog adds a log entry to the test results.
func (tr *TestResults) AddLog(log string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.Logs = append(tr.Logs, log)
}

// SetMetric sets a metric in the test results.
func (tr *TestResults) SetMetric(key string, value interface{}) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if tr.Metrics == nil {
		tr.Metrics = make(map[string]interface{})
	}
	tr.Metrics[key] = value
}

// IncrementResourceCount increments the count for a resource type.
func (tr *TestResults) IncrementResourceCount(resourceType string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if tr.ResourceCounts == nil {
		tr.ResourceCounts = make(map[string]int)
	}
	tr.ResourceCounts[resourceType]++
}

// TestSuite defines a collection of tests that can be run against a cloud provider.
type TestSuite struct {
	// Name is the name of the test suite.
	Name string

	// Description is the description of the test suite.
	Description string

	// Tests is the list of tests in the suite.
	Tests []Test

	// Setup is the setup function for the test suite.
	Setup func(TestInterface) error

	// Teardown is the teardown function for the test suite.
	Teardown func(TestInterface) error

	// Dependencies are the dependencies required for the test suite.
	Dependencies []string
}

// Test defines a single test that can be run against a cloud provider.
type Test struct {
	// Name is the name of the test.
	Name string

	// Description is the description of the test.
	Description string

	// Run is the function that runs the test.
	Run func(TestInterface) error

	// Skip determines whether to skip this test.
	Skip bool

	// SkipReason is the reason for skipping the test.
	SkipReason string

	// Timeout is the timeout for the test.
	Timeout time.Duration

	// Dependencies are the dependencies required for the test.
	Dependencies []string

	// Cleanup is the cleanup function for the test.
	Cleanup func(TestInterface) error
}

// TestRunner is responsible for running tests against cloud providers.
type TestRunner struct {
	// TestInterface is the test interface to use.
	TestInterface TestInterface

	// TestSuites are the test suites to run.
	TestSuites []TestSuite

	// Results are the results of the test execution.
	Results []TestResult

	// mu protects access to the TestRunner fields
	mu sync.RWMutex
}

// TestResult holds the result of a single test.
type TestResult struct {
	// Test is the test that was run.
	Test Test

	// Success indicates whether the test was successful.
	Success bool

	// Error is the error that occurred during the test.
	Error error

	// Duration is the duration of the test.
	Duration time.Duration

	// StartTime is the start time of the test.
	StartTime time.Time

	// EndTime is the end time of the test.
	EndTime time.Time
}

// NewTestRunner creates a new test runner.
func NewTestRunner(testInterface TestInterface) *TestRunner {
	return &TestRunner{
		TestInterface: testInterface,
		TestSuites:    []TestSuite{},
		Results:       []TestResult{},
	}
}

// AddTestSuite adds a test suite to the test runner.
func (tr *TestRunner) AddTestSuite(suite TestSuite) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.TestSuites = append(tr.TestSuites, suite)
}

// RunTests runs all the tests in the test runner.
func (tr *TestRunner) RunTests(ctx context.Context) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	for _, suite := range tr.TestSuites {
		if err := tr.runTestSuite(ctx, suite); err != nil {
			return fmt.Errorf("failed to run test suite %s: %w", suite.Name, err)
		}
	}

	return nil
}

// runTestSuite runs a single test suite.
func (tr *TestRunner) runTestSuite(ctx context.Context, suite TestSuite) error {
	// Run suite setup
	if suite.Setup != nil {
		if err := suite.Setup(tr.TestInterface); err != nil {
			return fmt.Errorf("failed to setup test suite %s: %w", suite.Name, err)
		}
	}

	// Run tests in the suite
	for _, test := range suite.Tests {
		if err := tr.runTest(ctx, test); err != nil {
			return fmt.Errorf("failed to run test %s in suite %s: %w", test.Name, suite.Name, err)
		}
	}

	// Run suite teardown
	if suite.Teardown != nil {
		if err := suite.Teardown(tr.TestInterface); err != nil {
			return fmt.Errorf("failed to teardown test suite %s: %w", suite.Name, err)
		}
	}

	return nil
}

// runTest runs a single test.
func (tr *TestRunner) runTest(ctx context.Context, test Test) error {
	// Skip test if requested
	if test.Skip {
		tr.Results = append(tr.Results, TestResult{
			Test:    test,
			Success: true, // Skipped tests are considered successful
		})
		return nil
	}

	// Set timeout for the test
	if test.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, test.Timeout)
		defer cancel()
	}

	// Run the test
	startTime := time.Now()
	err := test.Run(tr.TestInterface)
	endTime := time.Now()

	// Record the result
	result := TestResult{
		Test:      test,
		Success:   err == nil,
		Error:     err,
		Duration:  endTime.Sub(startTime),
		StartTime: startTime,
		EndTime:   endTime,
	}

	tr.Results = append(tr.Results, result)

	// Run cleanup if provided
	if test.Cleanup != nil {
		if cleanupErr := test.Cleanup(tr.TestInterface); cleanupErr != nil {
			// Log cleanup error but don't fail the test
			fmt.Printf("Warning: cleanup failed for test %s: %v\n", test.Name, cleanupErr)
		}
	}

	return err
}

// GetResults returns the results of the test execution.
func (tr *TestRunner) GetResults() []TestResult {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.Results
}

// GetSummary returns a summary of the test results.
func (tr *TestRunner) GetSummary() TestSummary {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	summary := TestSummary{
		TotalTests:    len(tr.Results),
		PassedTests:   0,
		FailedTests:   0,
		SkippedTests:  0,
		TotalDuration: 0,
	}

	for _, result := range tr.Results {
		summary.TotalDuration += result.Duration
		if result.Test.Skip {
			summary.SkippedTests++
		} else if result.Success {
			summary.PassedTests++
		} else {
			summary.FailedTests++
		}
	}

	return summary
}

// TestSummary holds a summary of test results.
type TestSummary struct {
	// TotalTests is the total number of tests run.
	TotalTests int

	// PassedTests is the number of tests that passed.
	PassedTests int

	// FailedTests is the number of tests that failed.
	FailedTests int

	// SkippedTests is the number of tests that were skipped.
	SkippedTests int

	// TotalDuration is the total duration of all tests.
	TotalDuration time.Duration
}

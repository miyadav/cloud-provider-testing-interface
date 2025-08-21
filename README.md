# Cloud Provider Testing Interface

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/miyadav/cloud-provider-testing-interface)

Reference [Idea](https://hackmd.io/@elmiko/ByfaBO4JJe)

This package provides a cloud-agnostic testing interface for Kubernetes cloud providers, similar to how `cloud.go` provides a common interface for cloud provider implementations. The testing interface allows cloud provider developers to write tests that are independent of the specific cloud provider implementation while ensuring consistent behavior across all cloud providers.

## Overview

The testing interface is designed to:

1. **Abstract Cloud Provider Details**: Provide a common interface that abstracts away the specific details of each cloud provider
2. **Ensure Consistent Testing**: Enable cloud providers to test the same functionality in a consistent way
3. **Simplify Test Development**: Reduce the complexity of writing tests for cloud providers
4. **Support Multiple Test Types**: Support unit tests, integration tests, and end-to-end tests
5. **Provide Test Infrastructure**: Offer common test utilities, mock implementations, and test runners

## Architecture

### Core Components

#### 1. TestInterface
The main interface that cloud provider test implementations must satisfy:

```go
type TestInterface interface {
    SetupTestEnvironment(config *TestConfig) error
    TeardownTestEnvironment() error
    GetCloudProvider() cloudprovider.Interface
    CreateTestNode(ctx context.Context, nodeConfig *TestNodeConfig) (*v1.Node, error)
    DeleteTestNode(ctx context.Context, nodeName string) error
    CreateTestService(ctx context.Context, serviceConfig *TestServiceConfig) (*v1.Service, error)
    DeleteTestService(ctx context.Context, serviceName string) error
    CreateTestRoute(ctx context.Context, routeConfig *TestRouteConfig) (*cloudprovider.Route, error)
    DeleteTestRoute(ctx context.Context, routeName string) error
    WaitForCondition(ctx context.Context, condition TestCondition) error
    GetTestResults() *TestResults
    ResetTestState() error
}
```

#### 2. TestRunner
Responsible for executing test suites and managing test results:

```go
type TestRunner struct {
    TestInterface TestInterface
    TestSuites    []TestSuite
    Results       []TestResult
}
```

#### 3. TestSuite
Defines a collection of related tests:

```go
type TestSuite struct {
    Name         string
    Description  string
    Tests        []Test
    Setup        func(TestInterface) error
    Teardown     func(TestInterface) error
    Dependencies []string
}
```

#### 4. Test
Defines a single test case:

```go
type Test struct {
    Name         string
    Description  string
    Run          func(TestInterface) error
    Skip         bool
    SkipReason   string
    Timeout      time.Duration
    Dependencies []string
    Cleanup      func(TestInterface) error
}
```

## Logic and Design Principles

### 1. Cloud-Agnostic Design
The interface is designed to be cloud-agnostic, meaning:
- Tests can be written without knowledge of specific cloud provider APIs
- Test logic focuses on Kubernetes resource behavior rather than cloud-specific implementation details
- Cloud provider implementations can be swapped without changing test code

### 2. Resource Lifecycle Management
The interface provides methods for creating and managing test resources:
- **Nodes**: Test node registration, addressing, and metadata
- **Services**: Test load balancer creation and management
- **Routes**: Test route creation and management

### 3. Test State Management
The interface includes comprehensive test state management:
- **Setup/Teardown**: Proper initialization and cleanup of test environments
- **Resource Tracking**: Automatic tracking of created resources for cleanup
- **Result Collection**: Structured collection of test results, metrics, and logs
- **State Reset**: Ability to reset test state between test runs

### 4. Condition-Based Testing
The interface supports condition-based testing for asynchronous operations:
- **WaitForCondition**: Wait for specific conditions to be met
- **Custom Check Functions**: Define custom logic for condition checking
- **Timeout Management**: Configurable timeouts for condition checking

### 5. Mock and Fake Implementations
The package provides mock and fake implementations for testing:
- **BaseTestImplementation**: Base implementation that can be extended
- **FakeTestImplementation**: Implementation using the fake cloud provider
- **MockClientBuilder**: Mock implementation of ControllerClientBuilder

## Usage Examples

### Basic Usage

```go
// Create a test implementation
fakeImpl := NewFakeTestImplementation()

// Create a test runner
runner := NewTestRunner(fakeImpl)

// Add test suites
runner.AddTestSuite(ExampleTestSuite())

// Run tests
ctx := context.Background()
err := runner.RunTests(ctx)
if err != nil {
    log.Fatalf("Test execution failed: %v", err)
}

// Get results
results := runner.GetResults()
summary := runner.GetSummary()
```

### Creating a Test Suite

```go
func MyTestSuite() TestSuite {
    return TestSuite{
        Name:        "My Cloud Provider Tests",
        Description: "Tests for my cloud provider implementation",
        Setup: func(ti TestInterface) error {
            config := &TestConfig{
                ProviderName:        "my-provider",
                ClusterName:         "test-cluster",
                Region:              "us-west-1",
                Zone:                "us-west-1a",
                TestTimeout:         5 * time.Minute,
                CleanupResources:    true,
                MockExternalServices: true,
            }
            return ti.SetupTestEnvironment(config)
        },
        Teardown: func(ti TestInterface) error {
            return ti.TeardownTestEnvironment()
        },
        Tests: []Test{
            {
                Name:        "Test Load Balancer",
                Description: "Tests load balancer functionality",
                Run:         testLoadBalancer,
                Timeout:     2 * time.Minute,
            },
        },
    }
}
```

### Writing Individual Tests

```go
func testLoadBalancer(ti TestInterface) error {
    ctx := context.Background()

    // Create test resources
    nodeConfig := &TestNodeConfig{
        Name:         "test-node",
        ProviderID:   "my-provider://test-node",
        InstanceType: "t3.medium",
        Zone:         "us-west-1a",
        Region:       "us-west-1",
    }

    node, err := ti.CreateTestNode(ctx, nodeConfig)
    if err != nil {
        return fmt.Errorf("failed to create test node: %w", err)
    }

    // Test cloud provider functionality
    cloud := ti.GetCloudProvider()
    loadBalancer, supported := cloud.LoadBalancer()
    if !supported {
        return fmt.Errorf("load balancer not supported")
    }

    // Perform tests...
    return nil
}
```

## Testing Steps and Logic

### Step 1: Environment Setup
1. **Initialize Test Configuration**: Set up provider-specific configuration
2. **Create Test Environment**: Initialize cloud provider and test infrastructure
3. **Set Up Mock Services**: Configure mock external services if needed
4. **Initialize Cloud Provider**: Call the cloud provider's Initialize method

### Step 2: Test Execution
1. **Create Test Resources**: Use the interface to create nodes, services, and routes
2. **Test Cloud Provider Methods**: Call cloud provider methods and verify results
3. **Verify Resource State**: Check that resources are in the expected state
4. **Test Error Conditions**: Verify proper error handling

### Step 3: Resource Cleanup
1. **Track Created Resources**: Automatically track all created resources
2. **Clean Up Resources**: Delete resources in the correct order
3. **Verify Cleanup**: Ensure all resources are properly cleaned up
4. **Reset Test State**: Reset the test environment for the next test

### Step 4: Result Collection
1. **Collect Test Results**: Gather test results, metrics, and logs
2. **Generate Test Summary**: Create a summary of test execution
3. **Report Failures**: Provide detailed information about test failures
4. **Store Test Artifacts**: Save test artifacts for analysis

## Integration with Existing Cloud Providers

### For New Cloud Providers
1. **Implement TestInterface**: Create a test implementation for your cloud provider
2. **Extend BaseTestImplementation**: Use the base implementation as a starting point
3. **Add Provider-Specific Tests**: Create tests specific to your cloud provider's features
4. **Register Test Suites**: Register your test suites with the test runner

### For Existing Cloud Providers
1. **Create Test Adapter**: Create an adapter that implements TestInterface
2. **Map to Existing Tests**: Map existing tests to the new interface
3. **Gradually Migrate**: Gradually migrate existing tests to use the new interface
4. **Maintain Compatibility**: Ensure existing test infrastructure continues to work

## Best Practices

### 1. Test Organization
- **Group Related Tests**: Organize tests into logical test suites
- **Use Descriptive Names**: Use clear, descriptive names for tests and test suites
- **Document Test Purpose**: Provide clear descriptions of what each test validates

### 2. Resource Management
- **Always Clean Up**: Ensure all test resources are properly cleaned up
- **Use Resource Tracking**: Leverage the built-in resource tracking for cleanup
- **Handle Cleanup Failures**: Handle cleanup failures gracefully

### 3. Error Handling
- **Test Error Conditions**: Include tests for error conditions and edge cases
- **Provide Clear Error Messages**: Use descriptive error messages for test failures
- **Handle Timeouts**: Use appropriate timeouts for asynchronous operations

### 4. Test Isolation
- **Independent Tests**: Ensure tests are independent and can run in any order
- **Reset State**: Reset test state between tests
- **Avoid Shared State**: Avoid sharing state between tests

### 5. Performance Considerations
- **Use Appropriate Timeouts**: Set reasonable timeouts for test operations
- **Limit Resource Creation**: Avoid creating unnecessary resources
- **Clean Up Promptly**: Clean up resources as soon as they're no longer needed

## Extending the Interface

### Adding New Resource Types
1. **Define Configuration**: Create a configuration struct for the new resource type
2. **Add Interface Methods**: Add create and delete methods to TestInterface
3. **Update Base Implementation**: Update BaseTestImplementation with default behavior
4. **Add Test Examples**: Create example tests for the new resource type

### Adding New Test Utilities
1. **Create Utility Functions**: Create utility functions for common test operations
2. **Add to Interface**: Add utility methods to TestInterface if needed
3. **Document Usage**: Document how to use the new utilities
4. **Add Examples**: Provide examples of using the new utilities

## Integration Guide for Cloud Provider Repositories

This section provides step-by-step instructions for integrating the cloud provider testing interface into your cloud provider repository.

### Step 1: Add the Dependency

Add the testing interface as a dependency to your cloud provider's `go.mod` file:

```bash
go get github.com/miyadav/cloud-provider-testing-interface
```

Or manually add to your `go.mod`:

```go
require (
    github.com/miyadav/cloud-provider-testing-interface v0.1.0
    // ... other dependencies
)
```

### Step 2: Create a Test Implementation

Create a test implementation that extends the base implementation for your specific cloud provider:

```go
// pkg/testing/cloud_provider_test_impl.go
package testing

import (
    "context"
    "fmt"
    
    "k8s.io/cloudprovider"
    testing "github.com/miyadav/cloud-provider-testing-interface"
)

// CloudProviderTestImplementation implements the testing interface for your cloud provider
type CloudProviderTestImplementation struct {
    *testing.BaseTestImplementation
    CloudProvider *cloudprovider.CloudProvider
    TestConfig    *testing.TestConfig
}

// NewCloudProviderTestImplementation creates a new test implementation
func NewCloudProviderTestImplementation(cloudProvider *cloudprovider.CloudProvider) *CloudProviderTestImplementation {
    baseImpl := testing.NewBaseTestImplementation(cloudProvider)
    return &CloudProviderTestImplementation{
        BaseTestImplementation: baseImpl,
        CloudProvider:          cloudProvider,
    }
}

// SetupTestEnvironment sets up the test environment for your cloud provider
func (c *CloudProviderTestImplementation) SetupTestEnvironment(config *testing.TestConfig) error {
    // Call the base implementation first
    if err := c.BaseTestImplementation.SetupTestEnvironment(config); err != nil {
        return err
    }
    
    // Add cloud provider-specific setup
    c.TestConfig = config
    
    // Initialize your cloud provider with test configuration
    if err := c.CloudProvider.Initialize(config.ClientBuilder, make(chan struct{})); err != nil {
        return fmt.Errorf("failed to initialize cloud provider: %w", err)
    }
    
    // Set up any cloud provider-specific test resources
    if err := c.setupCloudProviderResources(); err != nil {
        return fmt.Errorf("failed to setup cloud provider resources: %w", err)
    }
    
    return nil
}

// TeardownTestEnvironment cleans up the test environment
func (c *CloudProviderTestImplementation) TeardownTestEnvironment() error {
    // Clean up cloud provider-specific resources
    if err := c.cleanupCloudProviderResources(); err != nil {
        return fmt.Errorf("failed to cleanup cloud provider resources: %w", err)
    }
    
    // Call the base implementation
    return c.BaseTestImplementation.TeardownTestEnvironment()
}

// setupCloudProviderResources sets up cloud provider-specific test resources
func (c *CloudProviderTestImplementation) setupCloudProviderResources() error {
    // Implement cloud provider-specific resource setup
    // For example, create test VPCs, subnets, security groups, etc.
    return nil
}

// cleanupCloudProviderResources cleans up cloud provider-specific test resources
func (c *CloudProviderTestImplementation) cleanupCloudProviderResources() error {
    // Implement cloud provider-specific resource cleanup
    return nil
}
```

### Step 3: Create Test Suites

Create test suites that test your cloud provider's functionality:

```go
// pkg/testing/test_suites.go
package testing

import (
    "context"
    "fmt"
    "time"
    
    v1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/util/intstr"
    testing "github.com/miyadav/cloud-provider-testing-interface"
)

// CreateLoadBalancerTestSuite creates a test suite for load balancer functionality
func CreateLoadBalancerTestSuite() testing.TestSuite {
    return testing.TestSuite{
        Name:        "Load Balancer Tests",
        Description: "Tests load balancer creation, update, and deletion",
        Setup: func(ti testing.TestInterface) error {
            config := &testing.TestConfig{
                ProviderName:         "your-cloud-provider",
                ClusterName:          "test-cluster",
                Region:               "us-west-1",
                Zone:                 "us-west-1a",
                TestTimeout:          5 * time.Minute,
                CleanupResources:     true,
                MockExternalServices: false, // Use real cloud services for integration tests
            }
            return ti.SetupTestEnvironment(config)
        },
        Teardown: func(ti testing.TestInterface) error {
            return ti.TeardownTestEnvironment()
        },
        Tests: []testing.Test{
            {
                Name:        "Test Load Balancer Creation",
                Description: "Tests that a load balancer can be created successfully",
                Run:         testLoadBalancerCreation,
                Timeout:     2 * time.Minute,
            },
            {
                Name:        "Test Load Balancer Update",
                Description: "Tests that a load balancer can be updated",
                Run:         testLoadBalancerUpdate,
                Timeout:     2 * time.Minute,
            },
            {
                Name:        "Test Load Balancer Deletion",
                Description: "Tests that a load balancer can be deleted",
                Run:         testLoadBalancerDeletion,
                Timeout:     2 * time.Minute,
            },
        },
    }
}

// testLoadBalancerCreation tests load balancer creation
func testLoadBalancerCreation(ti testing.TestInterface) error {
    ctx := context.Background()
    
    // Create a test node
    nodeConfig := &testing.TestNodeConfig{
        Name:         "test-node-1",
        ProviderID:   "your-provider://test-node-1",
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
    serviceConfig := &testing.TestServiceConfig{
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
    
    // Test EnsureLoadBalancer
    nodes := []*v1.Node{node}
    status, err := loadBalancer.EnsureLoadBalancer(ctx, "test-cluster", service, nodes)
    if err != nil {
        return fmt.Errorf("failed to ensure load balancer: %w", err)
    }
    
    // Verify load balancer status
    if status == nil || len(status.Ingress) == 0 {
        return fmt.Errorf("load balancer status is empty")
    }
    
    ti.GetTestResults().AddLog("Load balancer creation test completed successfully")
    return nil
}

// testLoadBalancerUpdate tests load balancer update
func testLoadBalancerUpdate(ti testing.TestInterface) error {
    // Implement load balancer update test
    return nil
}

// testLoadBalancerDeletion tests load balancer deletion
func testLoadBalancerDeletion(ti testing.TestInterface) error {
    // Implement load balancer deletion test
    return nil
}
```

### Step 4: Create Integration Tests

Create integration tests that use your test implementation:

```go
// pkg/testing/integration_test.go
package testing

import (
    "context"
    "testing"
    
    "your-cloud-provider/pkg/cloudprovider"
    testing "github.com/miyadav/cloud-provider-testing-interface"
)

// TestLoadBalancerIntegration tests load balancer integration
func TestLoadBalancerIntegration(t *testing.T) {
    // Create your cloud provider instance
    cloudProvider := &cloudprovider.CloudProvider{}
    
    // Create test implementation
    testImpl := NewCloudProviderTestImplementation(cloudProvider)
    
    // Create test runner
    runner := testing.NewTestRunner(testImpl)
    
    // Add test suites
    runner.AddTestSuite(CreateLoadBalancerTestSuite())
    
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
```

### Step 5: Create a Test Runner Script

Create a script to run your tests:

```go
// cmd/test-runner/main.go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    
    "your-cloud-provider/pkg/cloudprovider"
    "your-cloud-provider/pkg/testing"
    testing "github.com/miyadav/cloud-provider-testing-interface"
)

func main() {
    var (
        testSuite = flag.String("suite", "all", "Test suite to run (all, loadbalancer, nodes, routes)")
        verbose   = flag.Bool("verbose", false, "Enable verbose output")
    )
    flag.Parse()
    
    // Create cloud provider instance
    cloudProvider := &cloudprovider.CloudProvider{}
    
    // Create test implementation
    testImpl := testing.NewCloudProviderTestImplementation(cloudProvider)
    
    // Create test runner
    runner := testing.NewTestRunner(testImpl)
    
    // Add test suites based on flag
    switch *testSuite {
    case "all":
        runner.AddTestSuite(testing.CreateLoadBalancerTestSuite())
        // Add other test suites
    case "loadbalancer":
        runner.AddTestSuite(testing.CreateLoadBalancerTestSuite())
    default:
        log.Fatalf("Unknown test suite: %s", *testSuite)
    }
    
    // Run tests
    ctx := context.Background()
    err := runner.RunTests(ctx)
    if err != nil {
        log.Fatalf("Test execution failed: %v", err)
    }
    
    // Get results
    results := runner.GetResults()
    summary := runner.GetSummary()
    
    // Print summary
    fmt.Printf("Test Summary:\n")
    fmt.Printf("  Total Tests: %d\n", summary.TotalTests)
    fmt.Printf("  Passed: %d\n", summary.PassedTests)
    fmt.Printf("  Failed: %d\n", summary.FailedTests)
    fmt.Printf("  Skipped: %d\n", summary.SkippedTests)
    fmt.Printf("  Duration: %v\n", summary.TotalDuration)
    
    // Print detailed results if verbose
    if *verbose {
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
    
    // Exit with error code if any tests failed
    if summary.FailedTests > 0 {
        os.Exit(1)
    }
}
```

### Step 6: Add to Your Build Pipeline

Add the test runner to your CI/CD pipeline:

```yaml
# .github/workflows/test.yml
name: Cloud Provider Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.24'
    
    - name: Install dependencies
      run: go mod download
    
    - name: Run unit tests
      run: go test -v ./pkg/...
    
    - name: Run integration tests
      run: go test -v ./pkg/testing/...
      env:
        CLOUD_PROVIDER_CONFIG: ${{ secrets.CLOUD_PROVIDER_CONFIG }}
    
    - name: Run test runner
      run: go run cmd/test-runner/main.go -suite=all -verbose
      env:
        CLOUD_PROVIDER_CONFIG: ${{ secrets.CLOUD_PROVIDER_CONFIG }}
```

### Step 7: Configuration Management

Create configuration files for different test environments:

```yaml
# config/test-config.yaml
test:
  provider:
    name: "your-cloud-provider"
    region: "us-west-1"
    zone: "us-west-1a"
  
  cluster:
    name: "test-cluster"
    version: "1.24.0"
  
  resources:
    cleanup: true
    timeout: "5m"
  
  external_services:
    mock: false  # Use real cloud services for integration tests
```

### Step 8: Documentation

Add documentation to your cloud provider repository:

```markdown
# Testing

This cloud provider uses the [cloud-provider-testing-interface](https://github.com/miyadav/cloud-provider-testing-interface) for comprehensive testing.

## Running Tests

### Unit Tests
```bash
go test -v ./pkg/...
```

### Integration Tests
```bash
go test -v ./pkg/testing/...
```

### Full Test Suite
```bash
go run cmd/test-runner/main.go -suite=all -verbose
```

## Test Configuration

Tests can be configured using environment variables or configuration files. See `config/test-config.yaml` for available options.

## Adding New Tests

1. Create a new test function in `pkg/testing/test_suites.go`
2. Add the test to an appropriate test suite
3. Update the test runner to include the new suite
4. Add integration tests in `pkg/testing/integration_test.go`
```

## Best Practices for Cloud Provider Integration

### 1. Test Environment Isolation
- Use separate test accounts/projects for each test run
- Implement proper cleanup to avoid resource leaks
- Use unique resource names to avoid conflicts

### 2. Configuration Management
- Use environment variables for sensitive configuration
- Provide default configurations for local development
- Validate configuration before running tests

### 3. Error Handling
- Implement proper error handling and logging
- Provide clear error messages for debugging
- Handle cloud provider API errors gracefully

### 4. Resource Management
- Track all created resources for cleanup
- Implement timeouts for resource operations
- Handle resource creation failures

### 5. Test Organization
- Group related tests into logical test suites
- Use descriptive test names and descriptions
- Implement test dependencies when needed

## Conclusion

The cloud provider testing interface provides a comprehensive, cloud-agnostic way to test cloud provider implementations. By abstracting away cloud-specific details and providing common test infrastructure, it enables cloud provider developers to focus on testing the behavior and functionality that should be consistent across all cloud providers.

The interface is designed to be extensible and can be adapted to support new resource types, test utilities, and testing patterns as needed. By following the design principles and best practices outlined in this document, cloud provider developers can create robust, maintainable tests that ensure their implementations work correctly and consistently.

## Getting Help

If you encounter issues while integrating the testing interface into your cloud provider:

1. Check the [examples](examples_test.go) for usage patterns
2. Review the [test implementations](implementation_test.go) for best practices
3. Open an issue in the [repository](https://github.com/miyadav/cloud-provider-testing-interface/issues)
4. Join the Kubernetes cloud provider community discussions
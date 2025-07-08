# Kola Testing Framework

Kola is Flatcar Container Linux's testing framework, designed to validate system integration across multiple cloud platforms and local environments. It provides a comprehensive suite of tools for testing OS functionality, container runtimes, and system services.

## Overview

Kola supports testing on:
- **Local/Virtual**: QEMU (full-featured), QEMU-unpriv (single-node)
- **Cloud Platforms**: AWS, Azure, GCE, DigitalOcean, Equinix Metal, OpenStack, ESX, and more

This documentation serves two primary audiences:
- **Test Writers** (External Contributors): Learn to write and run tests
- **Framework Maintainers**: Understand and extend Kola's architecture

---

# For Test Writers (External Contributors)

## Getting Started

### Requirements

**System Requirements:**
- Linux operating system
- SSH client
- **Sudo access for QEMU testing** (or use `qemu-unpriv` for single-node tests)
- Go 1.23+ for building from source

**Local Testing (QEMU):**
```bash
# Install required packages (Ubuntu/Debian)
sudo apt-get install -y qemu-kvm qemu-utils dnsmasq

# Install required packages (Fedora/RHEL)
sudo dnf install -y qemu-kvm qemu-img dnsmasq

# For WSL users, also install SeaBIOS
sudo apt-get install -y seabios
```

**For Cloud Testing:**
- Valid cloud provider credentials
- Appropriate IAM permissions for resource creation

### Quick Setup

1. **Clone and Build:**
```bash
git clone https://github.com/flatcar/mantle/
cd mantle
./build kola kolet
```

2. **Verify Installation:**
```bash
./bin/kola --help
./bin/kola list | head -10
```

**Alternative: Using Docker Container**
If you prefer not to build from source, you can use the pre-built container image:
```bash
# Use the container image with all dependencies included
sudo docker run --privileged --net host -v /dev:/dev --rm -it ghcr.io/flatcar/mantle:git-$(git rev-parse HEAD)
# Inside the container, kola is available in PATH and sudo is not needed:
# kola --help
# kola list | head -10
```

### Write Your First Test

Let's create a simple test to verify basic system functionality:

**1. Choose a Test Package**
Navigate to `kola/tests/` and find the appropriate package (e.g., `misc` for general tests).

**2. Add Your Test Function**
```go
// Test: Verify /etc/os-release exists and has content
func osReleaseExists(c cluster.TestCluster) {
    m := c.Machines()[0]
    output := c.MustSSH(m, "cat /etc/os-release")
    if len(output) == 0 {
        c.Errorf("/etc/os-release was empty")
    }
    c.Logf("os-release content length: %d bytes", len(output))
}
```

**3. Register Your Test**
Add to the package's `init()` function:
```go
func init() {
    register.Register(&register.Test{
        Run:         osReleaseExists,
        ClusterSize: 1,
        Name:        "misc.os-release-exists",
        Distros:     []string{"cl"}, // Flatcar Container Linux
    })
    // ...existing registrations...
}
```

**4. Build and Test**
```bash
./build kola
./bin/kola list | grep "os-release-exists"
```

### Run Tests Locally

**QEMU Testing (requires sudo for network setup):**
```bash
# Download a Flatcar image first
wget https://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_production_qemu_image.img

# Run your specific test
sudo ./bin/kola run -p qemu \
  --qemu-image ./flatcar_production_qemu_image.img \
  misc.os-release-exists

# Alternative: Single-node testing without sudo
./bin/kola run -p qemu-unpriv \
  --qemu-image ./flatcar_production_qemu_image.img \
  misc.os-release-exists

# For WSL users, if you get BIOS errors, specify firmware explicitly:
sudo ./bin/kola run -p qemu \
  --qemu-image ./flatcar_production_qemu_image.img \
  --qemu-firmware /usr/share/seabios/bios-256k.bin \
  misc.os-release-exists

# Run all tests in a package
sudo ./bin/kola run -p qemu \
  --qemu-image ./flatcar_production_qemu_image.img \
  "misc.*"
```

> **Note:** If you encounter `could not load PC BIOS 'bios-256k.bin'` errors, add `--qemu-firmware /usr/share/seabios/bios-256k.bin` to your command. Some systems may require installing the `seabios` package or using UEFI firmware instead.

**Useful Testing Options:**
```bash
# Run with verbose output (INFO level logging)
sudo ./bin/kola run -p qemu --verbose misc.os-release-exists

# Run with debug output (DEBUG level logging)
sudo ./bin/kola run -p qemu --debug misc.os-release-exists

# Run without destroying VMs (for debugging)
sudo ./bin/kola run -p qemu --remove=false misc.os-release-exists

# Set custom log level
sudo ./bin/kola run -p qemu --log-level=DEBUG misc.os-release-exists

# Increase SSH timeout for slow connections (default: 10s)
sudo ./bin/kola run -p qemu --ssh-timeout=30s misc.os-release-exists
```

## Kola Commands Reference

Kola provides several commands for different testing and development workflows:

### kola run

The primary command for executing tests. Supports comprehensive options for test execution, platforms, logging, and debugging.

**Basic usage:**
```bash
./bin/kola run -p qemu test.name
```

For a full list of available options and their descriptions, use the `--help` flag with any command (e.g., `./bin/kola run --help`).

See "Run Tests Locally" section above for detailed examples.

### kola list

Lists all available tests that can be executed.

```bash
./bin/kola list
./bin/kola list | grep podman    # Filter for specific tests
./bin/kola list | head -10       # Show first 10 tests
```

### kola spawn

Launches Flatcar Container Linux instances for interactive debugging and development.

```bash
# Spawn a single instance
./bin/kola spawn -p qemu

# Spawn multiple instances
./bin/kola spawn -p qemu --nodes=3
```

Use this command when you need to:
- Debug test failures interactively
- Explore the system manually
- Develop new tests with live feedback


## Grouping Tests

Sometimes it makes sense to group tests together under a specific package, especially when these tests are related and require the same test parameters. For `kola` it only takes a forwarding function to do testing groups. This forwarding function should take `cluster.TestCluster` as it's only input, and execute running other tests with `cluster.TestCluster.Run()`.

It is worth noting that the tests within the group are executed sequentially and on the same machine. As such, it is not recommended to group tests which modify the system state.

Additionally, the FailFast flag can be enabled during the test registration to skip any remaining steps after a failure has occurred.

Continuing with the look at the `podman` package we can see that `podman.base` is registered like so:

```golang
    register.Register(&register.Test{
            Run:         podmanBaseTest,
            ClusterSize: 1,
            Name:        `podman.base`,
            Distros:     []string{"rhcos"},
    })
```

If we look at `podmanBaseTest` it becomes very obvious that it's not a test of it's own, but a group of tests.

```go
func podmanBaseTest(c cluster.TestCluster) {
        c.Run("info", podmanInfo)
        c.Run("resources", podmanResources)
        c.Run("network", podmanNetworksReliably)
}
```

## Adding New Test Packages

If you need to add a new testing package, follow these steps:

**1. Create Package Directory:**
```bash
mkdir kola/tests/mypackage
```

**2. Create Package File:**
```bash
echo 'package mypackage' > kola/tests/mypackage/mypackage.go
```

**3. Register Package Import:**
Edit `kola/registry/registry.go` and add your package to the imports:
```go
import (
    _ "github.com/flatcar/mantle/kola/tests/coretest"
    _ "github.com/flatcar/mantle/kola/tests/mypackage"  // Add this line
    // ...existing imports...
)
```

**4. Implement Tests:**
Create your test implementation in `kola/tests/mypackage/mypackage.go`:
```go
package mypackage

import (
    "github.com/flatcar/mantle/kola/cluster"
    "github.com/flatcar/mantle/kola/register"
)

func init() {
    register.Register(&register.Test{
        Run:         myTestGroup,
        ClusterSize: 1,
        Name:        "mypackage.functionality",
        Distros:     []string{"cl"},
    })
}

func myTestGroup(c cluster.TestCluster) {
    c.Run("basic-test", myBasicTest)
    c.Run("advanced-test", myAdvancedTest)
}

func myBasicTest(c cluster.TestCluster) {
    m := c.Machines()[0]
    c.MustSSH(m, "echo 'Hello from my test'")
}

func myAdvancedTest(c cluster.TestCluster) {
    m := c.Machines()[0]
    output := c.MustSSH(m, "uname -r")
    c.Logf("Kernel version: %s", output)
}
```

### Test Grouping

Sometimes it makes sense to group tests together under a specific package, especially when these tests are related and require the same test parameters. For `kola` it only takes a forwarding function to do testing groups. This forwarding function should take `cluster.TestCluster` as it's only input, and execute running other tests with `cluster.TestCluster.Run()`.

It is worth noting that the tests within the group are executed sequentially and on the same machine. As such, it is not recommended to group tests which modify the system state.

Additionally, the FailFast flag can be enabled during the test registration to skip any remaining steps after a failure has occurred.

Continuing with the look at the `podman` package we can see that [`podman.base`](tests/podman/podman.go) is registered like so:

```golang
    register.Register(&register.Test{
            Run:         podmanBaseTest,
            ClusterSize: 1,
            Name:        `podman.base`,
            Distros:     []string{"rhcos"},
    })
```

If we look at `podmanBaseTest` it becomes very obvious that it's not a test of it's own, but a group of tests.

```go
func podmanBaseTest(c cluster.TestCluster) {
        c.Run("info", podmanInfo)
        c.Run("resources", podmanResources)
        c.Run("network", podmanNetworksReliably)
}
```

### Test Registration Details

Tests are registered using the `register.Register()` function in `kola/register/register.go`. The registration system supports platform filtering, distribution targeting, version constraints, and various configuration options.

For detailed registration options and examples, see the existing test packages in `kola/tests/` or refer to the `register.Test` struct definition in the source code.

### Parallelization and Performance

Kola supports running multiple tests concurrently using the `--parallel` flag, and tests can create multiple machines for parallel execution within a single test.

For performance optimization details and parallelization examples, see existing test implementations in `kola/tests/` packages.

### Native Code Execution

Kola supports running Go code directly on test machines via the kolet agent using "Native Functions". This allows executing complex logic without SSH overhead, useful for file operations, performance-critical code, or Go library integration.

For implementation details and examples, see the kolet documentation and existing native function usage in `kola/tests/` packages.

### Test Namespacing and Organization

**Hierarchical Naming:**
```go
// Package-based namespacing
"docker.basic"           // docker package, basic functionality  
"docker.networking"      // docker package, networking tests
"ignition.filesystems"   // ignition package, filesystem tests
"misc.kernel"            // misc package, kernel tests

// Feature-based sub-namespacing  
"cl.ignition.v3.files"   // Container Linux, Ignition v3, file operations
"cl.update.payload"      // Container Linux, update system, payload tests
```

**Best Practices:**
- **Use package prefixes**: Start with the package name
- **Be descriptive**: Names should clearly indicate test purpose
- **Group related tests**: Use consistent naming patterns
- **Avoid deep nesting**: Keep names reasonably short

**Registry Organization:**
The central registry (`kola/registry/registry.go`) imports all test packages:
```go
import (
    _ "github.com/flatcar/mantle/kola/tests/docker"
    _ "github.com/flatcar/mantle/kola/tests/ignition" 
    _ "github.com/flatcar/mantle/kola/tests/misc"
    // ... more packages
)
```

Tests are automatically discovered through Go's import side effects when packages are imported.

## Running Tests on Cloud Providers

### AWS
```bash
# Set up credentials <- The recommended way
export AWS_PROFILE=your-profile
# OR
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret

# Run tests
kola run -p aws \
  --aws-region us-west-2 \
  --aws-type t3.medium \
  misc.os-release-exists
```

### Azure
```bash  
# Set up credentials
az login

# Run tests
kola run -p azure \
  --azure-location westus2 \
  --azure-size Standard_D2s_v3 \
  misc.os-release-exists
```

### Google Cloud Platform
```bash
# Set up credentials
gcloud auth application-default login

# Run tests
kola run -p gce \
  --gce-project your-project-id \
  --gce-zone us-central1-f \
  --gce-machinetype n1-standard-2 \
  misc.os-release-exists
```

### Cost Management
- **Use smaller instance types** for basic tests
- **Set timeouts** to avoid runaway costs
- **Clean up resources** - kola handles this automatically on success however you should still verify
- **Manual cleanup** - Use the garbage collector for manual resource cleanup: `ore <platform> gc` (e.g., `ore aws gc`, `ore gce gc`)
- **Use QEMU locally** for development and basic validation

## Troubleshooting

### Common Issues

**SSH Connection Failures:**
```bash
# Enable debug output to see SSH details
sudo ./bin/kola run -p qemu --debug your.test

# Keep VMs running for manual inspection
sudo ./bin/kola run -p qemu --remove=false your.test

# SSH into running VMs for debugging (requires --remove=false and --key options)
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
    -o ProxyCommand="sudo nsenter -n -t <PID of QEMU instance> nc %h %p" \
    -p 22 core@<IP of QEMU instance>

# Enable VNC for visual debugging of boot issues
sudo ./bin/kola run -p qemu --qemu-vnc 0 --remove=false your.test
```

**QEMU Setup Issues:**
```bash
# Check if KVM is available
sudo kvm-ok

# Verify qemu installation
which qemu-system-x86_64

# Check permissions
sudo usermod -a -G kvm $USER
# (logout/login required)

# Alternative: Use qemu-unpriv if you prefer not to use sudo
./bin/kola run -p qemu-unpriv --qemu-image ./image.img your.test
```

**WSL/QEMU BIOS Issues:**
If you encounter `could not load PC BIOS '/path/to/bios-256k.bin'` errors:
```bash
# Find available BIOS/firmware files
find /usr/share -name "*bios*" -o -name "*seabios*" 2>/dev/null

# Use explicit firmware path (common locations)
sudo ./bin/kola run -p qemu --verbose \
  --qemu-image ./flatcar_production_qemu_image.img \
  --qemu-firmware /usr/share/seabios/bios-256k.bin \
  your.test

# Alternative: Use OVMF UEFI firmware if available
sudo ./bin/kola run -p qemu --verbose \
  --qemu-image ./flatcar_production_qemu_image.img \
  --qemu-firmware /usr/share/OVMF/OVMF_CODE.fd \
  your.test
```

**Test Timeouts:**
```bash
# Increase SSH connection timeout for slow environments
sudo ./bin/kola run -p qemu --ssh-timeout=30s your.test

# Increase SSH retry count (default is 60 retries)
sudo ./bin/kola run -p qemu --ssh-retries=100 your.test

# Check system resources
free -h
df -h
```

**Build Issues:**
```bash
# Clean and rebuild
make clean
./build kola

# Check Go version
go version  # Should be 1.23+
```

## Debugging and Logging

### Core Debugging Flags

**Debug Output:**
```bash
# Enable DEBUG level logging (most verbose)
sudo ./bin/kola run -p qemu --debug your.test

# Enable INFO level logging (verbose)
sudo ./bin/kola run -p qemu --verbose your.test

# Set specific log level
sudo ./bin/kola run -p qemu --log-level=DEBUG your.test
sudo ./bin/kola run -p qemu --log-level=INFO your.test
sudo ./bin/kola run -p qemu --log-level=NOTICE your.test  # default
sudo ./bin/kola run -p qemu --log-level=WARNING your.test
```

**VM Management:**
```bash
# Keep VMs running after test completion for inspection
sudo ./bin/kola run -p qemu --remove=false your.test

# Add custom SSH keys for debugging
sudo ./bin/kola run -p qemu --keys --key=/path/to/key.pub your.test
```

**Connection Tuning:**
```bash
# Increase SSH connection timeout (default: 10s)
sudo ./bin/kola run -p qemu --ssh-timeout=30s your.test

# Increase SSH retry attempts (default: 60)
sudo ./bin/kola run -p qemu --ssh-retries=100 your.test

# Platform-specific timeouts (for cloud platforms)
sudo ./bin/kola run -p equinixmetal --equinixmetal-launch-timeout=10m your.test
sudo ./bin/kola run -p equinixmetal --equinixmetal-install-timeout=20m your.test
```

**SystemD Debugging:**
```bash
# Enable debug logging for specific systemd units
# Note: Use the singular form --debug-systemd-unit multiple times for multiple units
sudo ./bin/kola run -p qemu --debug-systemd-unit=docker.service \
  --debug-systemd-unit=kubelet.service your.test
```

### Debug Output Interpretation

**Successful Test Output:**
```
=== RUN   misc.os-release-exists
2024/01/01 12:00:00 Creating machine for misc.os-release-exists
2024/01/01 12:00:30 Machine ready: 192.168.1.100
--- PASS: misc.os-release-exists (45.32s)
```

**Failed Test Output:**
```
=== RUN   misc.os-release-exists  
2024/01/01 12:00:00 Creating machine for misc.os-release-exists
2024/01/01 12:00:30 Machine ready: 192.168.1.100
misc.os-release-exists: /etc/os-release was empty
--- FAIL: misc.os-release-exists (45.32s)
```

### Platform-Specific Gotchas

**QEMU:**
- **Requires sudo for network setup** - Kola creates a virtual network bridge for VM communication (use `qemu-unpriv` for single-node tests without sudo)
- **Enable IPv4 forwarding** - Allows VMs to access the internet through your host: `sudo sysctl -w net.ipv4.ip_forward=1`
- **Automatic iptables rules** - Kola creates NAT rules so VMs can reach external services (docker registry, package repos, etc.)
- **Disable conflicting firewalls** - Services like `firewalld` can block VM network traffic: `sudo systemctl stop firewalld.service` (permanent: `sudo systemctl disable --now firewalld.service`)
- **Host resources matter** - Performance varies with available CPU cores, RAM, and KVM hardware acceleration
- **Network troubleshooting** - If VMs can't reach the internet, check firewall rules and IP forwarding settings

**Cloud Platforms:**
- Check quotas and limits before running large test suites
- Verify regions support your instance types
- Some platforms have minimum billing increments

---

# For Framework Maintainers

## Architecture Overview

Kola's architecture follows a three-tier abstraction model:

```
Flight (Platform Setup)
    ↓
Cluster (Test Environment) 
    ↓
Machine (Test Targets)
```

### Core Concepts

**Flight**: Platform-specific test environment setup
- Manages cloud provider authentication and configuration
- Handles image selection and instance type configuration
- Coordinates resource lifecycle (creation, cleanup)
- Abstracts platform differences behind common interface

**Cluster**: Collection of machines for testing
- Manages multiple machine instances as a group
- Provides SSH connectivity and command execution
- Handles test orchestration and synchronization
- Manages machine lifecycle within tests

**Machine**: Individual test instances
- Represents a single VM or container instance
- Provides platform-agnostic machine operations
- Handles platform-specific machine management
- Exposes common properties (IP, ID, etc.)

### Component Relationships

```
┌─────────────────────────────────────────────────────────┐
│                         Flight                          │
│  ┌───────────────────────────────────────────────────┐  │
│  │                      Cluster                      │  │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐  │  │
│  │  │   Machine   │ │   Machine   │ │   Machine   │  │  │
│  │  │     #1      │ │     #2      │ │     #N      │  │  │
│  │  └─────────────┘ └─────────────┘ └─────────────┘  │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Key Abstractions

### Platform Interface

The platform interface (`platform/platform.go`) provides a common abstraction across all supported cloud providers and local environments:

```go
type Platform interface {
    // Platform identification
    Name() string
    
    // Machine lifecycle
    NewMachine(userdata *conf.UserData) (Machine, error)
    
    // Cluster operations  
    NewCluster(opts *ClusterOptions) (Cluster, error)
    
    // Resource management
    Destroy()
}
```

**Platform Implementations:**
- **QEMU**: Local virtual machines via qemu-kvm
- **AWS**: EC2 instances with VPC networking  
- **Azure**: Virtual machines with resource groups
- **GCE**: Compute Engine instances with networks
- **And more**: Each implementing the common interface

### Test Registration System

The registration system [`kola/register/register.go`](register/register.go) manages test discovery, filtering, and execution:

**Test Structure:**
```go
type Test struct {
    Name        string          // Unique test identifier
    Run         func(TestCluster) // Test implementation
    ClusterSize int             // Number of machines needed
    
    // Filtering
    Platforms        []string   // Allowed platforms
    ExcludePlatforms []string   // Excluded platforms  
    Distros         []string   // Target distributions
    
    // Configuration
    Flags           []Flag     // Test behavior flags
    Timeout         time.Duration // Test timeout
    UserData        *conf.UserData // Cloud-init config
}
```

**Registry Pattern:**
Tests are automatically discovered through Go's import side effects:
1. Test packages register tests in their `init()` functions
2. `kola/registry/registry.go` imports all test packages
3. Registration happens at package import time
4. Tests are filtered and executed by the main harness

### SSH Execution Framework

SSH command execution is handled through the cluster interface and serves as the primary mechanism for interacting with test machines across all platforms (local QEMU, cloud providers, etc.):

**Command Execution Patterns:**
```go
// Basic execution with error handling
output, err := cluster.SSH(machine, "command")

// Must succeed - fails test on error  
output := cluster.MustSSH(machine, "command")

// Assert output contains expected text
cluster.AssertCmdOutputContains(machine, "command", "expected")
```

**Implementation Details:**
- **Connection Pooling**: SSH connections are reused per machine; each command creates a new session on the existing connection
- **Output Handling**: Commands capture both stdout and stderr; stderr appears in test logs, all output in debug logs
- **Error Types**: Distinguishes between SSH connection failures and command execution failures (non-zero exit codes)
- **Timeouts**: Connection establishment, individual commands, and overall tests all have timeout protection
- **Performance**: First SSH call per machine has setup overhead (~100-500ms), subsequent calls are faster (~10-50ms)

## Extending the Framework

### Adding New Platforms

To add support for a new cloud platform:

1. **Implement Platform Interface** (`platform/newcloud/`):
```go
type NewCloudPlatform struct {
    // Platform-specific configuration
}

func (p *NewCloudPlatform) NewMachine(userdata *conf.UserData) (Machine, error) {
    // Create instance using cloud provider API
}

func (p *NewCloudPlatform) NewCluster(opts *ClusterOptions) (Cluster, error) {
    // Create cluster of instances
}
```

2. **Add Platform Registration** (`platform/platform.go`):
```go
func init() {
    RegisterPlatform("newcloud", &NewCloudPlatform{})
}
```

3. **Add Command-Line Support** (`cmd/kola/kola.go`):
```go
// Add platform-specific flags
var (
    newcloudRegion     = flag.String("newcloud-region", "", "NewCloud region")
    newcloudInstanceType = flag.String("newcloud-type", "", "Instance type")
)
```

4. **Update Documentation** and add example usage

### Implementing New Test Utilities

**Add TestCluster Methods** (`kola/cluster/cluster.go`):
```go
func (t TestCluster) NewUtilityMethod(machine platform.Machine, args string) error {
    // Implement new testing utility
    return t.SSH(machine, fmt.Sprintf("utility-command %s", args))
}
```

**Add Native Functions** (via kolet agent):
1. Implement function in `cmd/kolet/kolet.go`
2. Add RPC handler for network calls
3. Add TestCluster method to invoke via `RunNative`

### Modifying Test Execution Flow

**Test Harness** (`kola/harness.go`):
- Controls overall test execution flow
- Manages platform setup and teardown  
- Handles filtering and test selection
- Coordinates parallel test execution

**Key Extension Points:**
- **Pre-test setup**: Platform configuration, image preparation
- **Test execution**: Machine creation, SSH setup, test invocation
- **Post-test cleanup**: Resource destruction, log collection

## Development Workflow

### Building and Testing Changes

**Local Development:**
```bash
# Build specific components
./build kola           # Build test runner
./build kolet          # Build test agent  
./build ore            # Build cloud utilities

# Test with framework changes
sudo ./bin/kola run -p qemu simple-test

# Test with detailed logging
sudo ./bin/kola run -p qemu --debug complex-test
```

**Integration Testing:**
```bash
# Test locally first (always start here)
sudo ./bin/kola run -p qemu --qemu-image ./flatcar_image.img basic-suite

# Test on cloud platforms with proper configuration
# AWS (requires credentials and region)
./bin/kola run -p aws --aws-region us-west-2 --aws-type t3.medium basic-suite

# GCE (requires project and credentials)  
./bin/kola run -p gce --gce-project my-project --gce-zone us-central1-f basic-suite

# Azure (requires location and credentials)
./bin/kola run -p azure --azure-location westus2 --azure-size Standard_D2s_v3 basic-suite

# Parallel execution for faster testing (runs tests concurrently on multiple instances)
./bin/kola run -p qemu --parallel 4 "misc.*"
```

### Code Organization

**Key Directories:**
- `kola/` - Main testing framework
  - `harness.go` - Test execution engine
  - `cluster/` - Test cluster management  
  - `register/` - Test registration system
  - `tests/` - All test implementations
- `platform/` - Platform abstraction layer
- `cmd/kola/` - Command-line interface
- `cmd/kolet/` - Test agent for machines

**Important Files:**
- `kola/harness.go` - Core test harness
- `kola/register/register.go` - Test registration
- `kola/cluster/cluster.go` - Test cluster interface
- `platform/platform.go` - Platform abstraction
- `cmd/kola/kola.go` - Main CLI entry point

### Performance Considerations

**Test Execution:**
- Tests run in parallel where possible
- SSH connections are pooled and reused
- Machine creation is optimized per platform
- Resource cleanup is automatic and robust

**Platform Optimization:**
- Cloud provider APIs are used efficiently
- Instance types are chosen for test requirements
- Network setup minimizes latency
- Storage is optimized for test data

**Scaling:**
- Test suite scales to hundreds of tests
- Parallel execution configurable per platform
- Resource limits prevent runaway usage
- Timeouts ensure tests complete reliably

---

# Reference Documentation

## Test Package Organization

**Current Test Packages:**
- `coretest/` - Core OS functionality
- `docker/` - Docker container runtime
- `podman/` - Podman container runtime  
- `systemd/` - systemd service management
- `ignition/` - Ignition configuration
- `update/` - OS update mechanisms
- `etcd/` - etcd cluster functionality
- `misc/` - General system tests


## Contributing

1. **Fork** the [mantle repository](https://github.com/flatcar/mantle/)
2. **Create** a feature branch for your changes
3. **Write** tests following the patterns in this guide
4. **Test** locally with QEMU before submitting
5. **Submit** a pull request with clear description

**Testing Your Changes:**
- Run existing tests to ensure no regressions
- Add tests for new functionality  
- Test on multiple platforms when possible
- Update documentation for new features

For questions or support, please open an issue in the [Flatcar repository](https://github.com/flatcar/flatcar/issues).

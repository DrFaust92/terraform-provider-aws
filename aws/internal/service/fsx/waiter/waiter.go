package waiter

import (
	"time"

	"github.com/aws/aws-sdk-go/service/fsx"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	FileSystemAvailableDelay = 30 * time.Second
	FileSystemDeletedDelay   = 30 * time.Second
)

// FileSystemAvailable waits for a FileSystem to return Available
func FileSystemAvailable(conn *fsx.FSx, id string, timeout time.Duration) (*fsx.FileSystem, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{fsx.FileSystemLifecycleUpdating, fsx.FileSystemLifecycleCreating},
		Target:  []string{fsx.FileSystemLifecycleAvailable},
		Refresh: FileSystemStatus(conn, id),
		Timeout: timeout,
		Delay:   FileSystemAvailableDelay,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*fsx.FileSystem); ok {
		return output, err
	}

	return nil, err
}

// FileSystemDeleted waits for a FileSystem to return Deleted
func FileSystemDeleted(conn *fsx.FSx, id string, timeout time.Duration) (*fsx.FileSystem, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{fsx.FileSystemLifecycleAvailable, fsx.FileSystemLifecycleDeleting},
		Target:  []string{},
		Refresh: FileSystemStatus(conn, id),
		Timeout: timeout,
		Delay:   FileSystemDeletedDelay,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*fsx.FileSystem); ok {
		return output, err
	}

	return nil, err
}

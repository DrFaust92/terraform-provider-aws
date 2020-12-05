package waiter

import (
	"time"

	"github.com/aws/aws-sdk-go/service/fsx"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	FileSystemAvailableDelay                                  = 30 * time.Second
	FileSystemDeletedDelay                                    = 30 * time.Second
	FileSystemWindowsAliasAvailableTimeout                    = 5 * time.Minute
	FileSystemWindowsAliasDeletedTimeout                      = 5 * time.Minute
	FileSystemAdministrativeActionsCompletedOrOptimizingDelay = 30 * time.Second
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

// FileSystemWindowsAliasAvailable waits for a File System Windows Alias to return Available
func FileSystemWindowsAliasAvailable(conn *fsx.FSx, id, aliasID string) (*fsx.Alias, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{fsx.AliasLifecycleCreating},
		Target:  []string{fsx.AliasLifecycleAvailable},
		Refresh: FileSystemWindowsAliasStatus(conn, id, aliasID),
		Timeout: FileSystemWindowsAliasAvailableTimeout,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*fsx.Alias); ok {
		return output, err
	}

	return nil, err
}

// FileSystemWindowsAliasDeleted waits for a File System Windows Alias to return Deleted
func FileSystemWindowsAliasDeleted(conn *fsx.FSx, id, aliasID string) (*fsx.Alias, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{fsx.AliasLifecycleDeleting},
		Target:  []string{},
		Refresh: FileSystemWindowsAliasStatus(conn, id, aliasID),
		Timeout: FileSystemWindowsAliasDeletedTimeout,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*fsx.Alias); ok {
		return output, err
	}

	return nil, err
}

// FileSystemAdministrativeActionsCompletedOrOptimizing waits for a File System Administrative Actions to return Completed or Updated Optimizing
func FileSystemAdministrativeActionsCompletedOrOptimizing(conn *fsx.FSx, id, action string, timeout time.Duration) (*fsx.FileSystem, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			fsx.StatusInProgress,
			fsx.StatusPending,
		},
		Target: []string{
			fsx.StatusCompleted,
			fsx.StatusUpdatedOptimizing,
		},
		Refresh: FileSystemAdministrativeActionsStatus(conn, id, action),
		Timeout: timeout,
		Delay:   FileSystemAvailableDelay,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*fsx.FileSystem); ok {
		return output, err
	}

	return nil, err
}

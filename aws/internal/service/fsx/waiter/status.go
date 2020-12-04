package waiter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/fsx"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/fsx/finder"
)

const (
	FileSystemStatusNotFound             = "NotFound"
	FileSystemStatusUnknown              = "Unknown"
	FileSystemWindowsAliasStatusNotFound = "NotFound"
)

// FileSystemStatus fetches the File System and its Lifecycle
func FileSystemStatus(conn *fsx.FSx, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := finder.FileSystemByID(conn, id)
		if tfawserr.ErrCodeEquals(err, fsx.ErrCodeFileSystemNotFound) {
			return nil, FileSystemStatusNotFound, nil
		}

		if err != nil {
			return nil, FileSystemStatusUnknown, err
		}

		if output == nil {
			return nil, FileSystemStatusNotFound, nil
		}

		return output, aws.StringValue(output.Lifecycle), nil
	}
}

// FileSystemWindowsAliasStatus fetches the Windows File System Alias and its Lifecycle
func FileSystemWindowsAliasStatus(conn *fsx.FSx, id, aliasID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := finder.FileSystemByID(conn, id)
		if tfawserr.ErrCodeEquals(err, fsx.ErrCodeFileSystemNotFound) {
			return nil, FileSystemStatusNotFound, nil
		}

		if err != nil {
			return nil, FileSystemStatusUnknown, err
		}

		if output == nil {
			return nil, FileSystemStatusNotFound, nil
		}

		winConfig := output.WindowsConfiguration
		if output.WindowsConfiguration == nil {
			return nil, FileSystemStatusUnknown, err
		}

		aliases := winConfig.Aliases
		if aliases == nil || len(aliases) == 0 {
			return nil, FileSystemWindowsAliasStatusNotFound, nil
		}

		var lifecycle string
		for _, a := range aliases {
			if aws.StringValue(a.Name) == aliasID {
				lifecycle = aws.StringValue(a.Lifecycle)
				break
			}
		}

		return aliasID, lifecycle, nil
	}
}

// FileSystemAdministrativeActionsStatus fetches the  File System and its Status
func FileSystemAdministrativeActionsStatus(conn *fsx.FSx, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := finder.FileSystemByID(conn, id)
		if tfawserr.ErrCodeEquals(err, fsx.ErrCodeFileSystemNotFound) {
			return nil, FileSystemStatusNotFound, nil
		}

		if err != nil {
			return nil, FileSystemStatusUnknown, err
		}

		if output == nil {
			return nil, FileSystemStatusNotFound, nil
		}

		for _, administrativeAction := range output.AdministrativeActions {
			if administrativeAction == nil {
				continue
			}

			if aws.StringValue(administrativeAction.AdministrativeActionType) == fsx.AdministrativeActionTypeFileSystemUpdate {
				return output, aws.StringValue(administrativeAction.Status), nil
			}
		}

		return output, fsx.StatusCompleted, nil
	}
}

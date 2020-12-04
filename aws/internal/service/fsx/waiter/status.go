package waiter

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/fsx"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	FileSystemStatusNotFound = "NotFound"
	FileSystemStatusUnknown  = "Unknown"
)

// FileSystemStatus fetches the Volume and its Status
func FileSystemStatus(conn *fsx.FSx, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &fsx.DescribeFileSystemsInput{
			FileSystemIds: []*string{aws.String(id)},
		}

		var filesystem *fsx.FileSystem

		err := conn.DescribeFileSystemsPages(input, func(page *fsx.DescribeFileSystemsOutput, lastPage bool) bool {
			for _, fs := range page.FileSystems {
				if aws.StringValue(fs.FileSystemId) == id {
					filesystem = fs
					return false
				}
			}

			return !lastPage
		})

		if tfawserr.ErrCodeEquals(err, fsx.ErrCodeFileSystemNotFound) {
			return nil, FileSystemStatusNotFound, nil
		}

		if err != nil {
			return nil, FileSystemStatusUnknown, err
		}

		if filesystem == nil {
			return nil, FileSystemStatusNotFound, nil
		}

		return filesystem, aws.StringValue(filesystem.Lifecycle), nil
	}
}

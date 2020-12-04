package finder

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/fsx"
)

// FileSystemByID returns the FileSystem corresponding to the specified ID.
func FileSystemByID(conn *fsx.FSx, id string) (*fsx.FileSystem, error) {
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

	if err != nil {
		return nil, err
	}

	return filesystem, nil
}

package distributor

import "github.com/theoptz/basic-s3/proto"

type Distributor interface {
	GetPlan(fileSize int) (clients []int, size int)
	GetClientByID(id int) (proto.StorageClient, error)
}

package client

import (
	"github.com/viciious/mika/config"
	pb "github.com/viciious/mika/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func New() (pb.MikaClient, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	c, err := grpc.Dial(config.API.Listen, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to dial tracker")
	}
	return pb.NewMikaClient(c), nil
}

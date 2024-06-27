package rpc

import (
	"context"
	"github.com/jedib0t/go-pretty/v6/table"
	pb "github.com/viciious/mika/proto"
	"github.com/viciious/mika/store"
	"github.com/viciious/mika/tracker"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"os"
)

type MikaService struct {
	pb.UnimplementedMikaServer
}

func PBToWhiteList(p *pb.WhiteList) *store.WhiteListClient {
	return &store.WhiteListClient{
		ClientPrefix: p.Prefix,
		ClientName:   p.Name,
	}
}

func WhiteListToPB(w *store.WhiteListClient) *pb.WhiteList {
	return &pb.WhiteList{
		Prefix: w.ClientPrefix,
		Name:   w.ClientName,
	}
}

func renderWhiteList(wl []*store.WhiteListClient, title string) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	if title != "" {
		t.SetTitle(title)
	}
	t.AppendHeader(table.Row{"name", "prefix"})
	for _, w := range wl {
		t.AppendRow(table.Row{w.ClientName, w.ClientPrefix})
	}
	t.SortBy([]table.SortBy{{
		Name: "name",
	}})
	t.Render()
}

func (s *MikaService) ConfigAll(context.Context, *emptypb.Empty) (*pb.ConfigAllResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConfigAll not implemented")
}
func (s *MikaService) ConfigSave(context.Context, *pb.ConfigSaveParams) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConfigSave not implemented")
}

func (s *MikaService) WhiteListAdd(_ context.Context, params *pb.WhiteList) (*emptypb.Empty, error) {
	wl := &store.WhiteListClient{ClientPrefix: params.Prefix, ClientName: params.Name}
	err := tracker.WhiteListAdd(wl)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add whitelist client")
	}
	renderWhiteList([]*store.WhiteListClient{wl}, "Whitelisted Client")
	log.Infof("Added new whitelisted client: %s", params.Name)
	return &emptypb.Empty{}, nil
}

func (s *MikaService) WhiteListDelete(_ context.Context, params *pb.WhiteListDeleteParams) (*emptypb.Empty, error) {
	w, err := tracker.WhiteListGet(params.Prefix)
	if err != nil {
		return &emptypb.Empty{}, status.Errorf(codes.NotFound, "unknown client prefix")
	}
	if err := tracker.WhiteListDelete(w); err != nil {
		return &emptypb.Empty{}, status.Errorf(codes.NotFound, "error removing client from whitelist")
	}
	return &emptypb.Empty{}, nil
}

func (s *MikaService) WhiteListAll(context.Context, *emptypb.Empty) (*pb.WhiteListAllResponse, error) {
	var wl []*pb.WhiteList
	for _, wlc := range tracker.WhiteList() {
		wl = append(wl, WhiteListToPB(wlc))
	}
	return &pb.WhiteListAllResponse{Whitelists: wl}, nil
}

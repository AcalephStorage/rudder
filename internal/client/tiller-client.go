package client

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	tiller "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/version"
)

type TillerClient struct {
	address string
	context context.Context
}

func NewTillerClient(address string) *TillerClient {
	md := metadata.Pairs("x-helm-api-client", version.Version)
	ctx := metadata.NewContext(context.TODO(), md)
	return &TillerClient{address: address, context: ctx}
}

func (tc *TillerClient) execute(request func(tiller.ReleaseServiceClient)) error {
	conn, err := grpc.Dial(tc.address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	rsc := tiller.NewReleaseServiceClient(conn)
	request(rsc)
	return nil
}

func (tc *TillerClient) ListReleases(req *tiller.ListReleasesRequest) (res *tiller.ListReleasesResponse, err error) {
	tc.execute(func(rsc tiller.ReleaseServiceClient) {
		lrc, err := rsc.ListReleases(tc.context, req)
		if err != nil {
			return
		}
		res, err = lrc.Recv()
	})
	return
}

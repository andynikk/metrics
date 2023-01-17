package api

import (
	"context"
	"fmt"

	"github.com/andynikk/advancedmetrics/internal/general"
	"github.com/andynikk/advancedmetrics/internal/grpchandlers"
	"google.golang.org/grpc/metadata"
)

type GRPCServer struct {
	RepStore general.RepStore[grpchandlers.RepStore]
}

func FillHeader(ctx context.Context) general.Header {
	mHeader := make(general.Header)

	mdI, _ := metadata.FromIncomingContext(ctx)
	for key, valMD := range mdI {
		for _, val := range valMD {
			mHeader[key] = val
		}
	}

	mdO, _ := metadata.FromOutgoingContext(ctx)
	for key, valMD := range mdO {
		for _, val := range valMD {
			mHeader[key] = val
		}
	}

	return mHeader
}

func (s *GRPCServer) mustEmbedUnimplementedMetricCollectorServer() {
	//TODO implement me
	panic("implement me")
}

func (s *GRPCServer) UpdatesAllMetricsJSON(ctx context.Context, req *UpdatesRequest) (*TextErrResponse, error) {
	header := FillHeader(ctx)

	err := s.RepStore.HandlerUpdatesMetricJSON(header, req.Body)
	if err != nil {
		return &TextErrResponse{Result: []byte(err.Error())}, err
	}
	return &TextErrResponse{Result: []byte("")}, nil
}

func (s *GRPCServer) UpdateOneMetricsJSON(ctx context.Context, req *UpdateStrRequest) (*TextErrResponse, error) {
	header := FillHeader(ctx)

	err := s.RepStore.HandlerUpdateMetricJSON(header, req.Body)
	if err != nil {
		return &TextErrResponse{Result: []byte(err.Error())}, err
	}
	return &TextErrResponse{Result: []byte("")}, nil
}

func (s *GRPCServer) UpdateOneMetrics(ctx context.Context, req *UpdateRequest) (*TextErrResponse, error) {
	err := s.RepStore.HandlerSetMetricaPOST(string(req.MetType), string(req.MetName), string(req.MetValue))

	if err != nil {
		return &TextErrResponse{Result: []byte(err.Error())}, err
	}

	return &TextErrResponse{Result: []byte("")}, nil
}

func (s *GRPCServer) PingDataBases(ctx context.Context, req *EmptyRequest) (*TextErrResponse, error) {
	header := FillHeader(ctx)

	err := s.RepStore.HandlerPingDB(header)
	if err != nil {
		return &TextErrResponse{Result: []byte(err.Error())}, err
	}
	return &TextErrResponse{Result: []byte("")}, nil
}

func (s *GRPCServer) GetValueJSON(ctx context.Context, req *UpdatesRequest) (*FullResponse, error) {
	header := FillHeader(ctx)

	h, body, err := s.RepStore.HandlerValueMetricaJSON(header, req.Body)
	ok := true
	if err != nil {
		ok = false
	}

	var hdr string
	for k, v := range h {
		hdr += fmt.Sprintf("%s:%s\n", k, v)
	}

	return &FullResponse{Header: []byte(hdr), Body: body, Result: ok}, err
}

func (s *GRPCServer) GetValue(ctx context.Context, req *UpdatesRequest) (*StatusResponse, error) {
	val, err := s.RepStore.HandlerGetValue(req.Body)
	return &StatusResponse{Result: []byte(val)}, err
}

func (s *GRPCServer) GetListMetrics(ctx context.Context, req *EmptyRequest) (*StatusResponse, error) {
	header := FillHeader(ctx)
	_, val := s.RepStore.HandlerGetAllMetrics(header)
	return &StatusResponse{Result: val}, nil
}

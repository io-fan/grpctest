package server

import (
	"context"
	pb "grpctest/proto"
	"log"
	"net"
	"strconv"
	"time"

	"google.golang.org/grpc"
)

type StreamService struct{}

const (
	Address string = ":8000"
	Network string = "tcp"
)

func main() {
	listener, err := net.Listen(Network, Address)
	if err != nil {
		log.Fatalf("net listen error: %v", err)
	}
	grpcServer := grpc.NewServer()
	//pb.RegisterSimpleServer(grpcServer, &SimpleService{})
	pb.RegisterStreamServerServer(grpcServer, &StreamService{})
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("grpcserver.serve err:%v", err)
	}
}

func (s *StreamService) Route(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {
	res := pb.SimpleResponse{
		Code:  200,
		Value: "hello," + req.Data,
	}
	return &res, nil
}

func (s *StreamService) ListValue(req *pb.SimpleRequest, srv pb.StreamServer_ListValueServer) error {
	for n := 0; n < 5; n++ {
		err := srv.Send(&pb.StreamResponse{
			StreamValue: req.Data + strconv.Itoa(n),
		})
		if err != nil {
			return err
		}
		time.Sleep(time.Second * 1)
	}
	return nil
}

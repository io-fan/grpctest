package main

import (
	"context"
	pb "grpctest/proto"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"google.golang.org/grpc"
)

type AllService struct{}

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
	pb.RegisterAllServiceServer(grpcServer, &AllService{})
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("grpcserver.serve err:%v", err)
	}
}

func (s *AllService) Route(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {
	res := pb.SimpleResponse{
		Code:  200,
		Value: "hello," + req.Data,
	}
	return &res, nil
}

func (s *AllService) ListValue(req *pb.SimpleRequest, srv pb.AllService_ListValueServer) error {
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

func (s *AllService) RouteList(srv pb.AllService_RouteListServer) error {
	for {
		res, err := srv.Recv()
		if err == io.EOF {
			return srv.SendAndClose(&pb.SimpleResponse{Code: 200, Value: "ok"})
		}
		if err != nil {
			return err
		}
		log.Println(res.StreamData)
	}
}

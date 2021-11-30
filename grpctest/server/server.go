package main

import (
	"context"
	pb "grpctest/proto"
	"io"
	"log"
	"net"
	"runtime"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	data := make(chan *pb.SimpleResponse, 1)
	go handle(ctx, req, data)
	select {
	case res := <-data:
		return res, nil
	case <-ctx.Done():
		return nil, status.Errorf(codes.Canceled, "client canceled.abandoning")
	}
}

func handle(ctx context.Context, req *pb.SimpleRequest, data chan<- *pb.SimpleResponse) {
	select {
	case <-ctx.Done():
		log.Println(ctx.Err())
		runtime.Goexit() //退出协程
	case <-time.After(4 * time.Second):
		res := pb.SimpleResponse{
			Code:  200,
			Value: "hello," + req.Data,
		}
		//if ctx.err()==context.canceled

		data <- &res
	}

}

func (s *AllService) ListValue(req *pb.SimpleRequest, srv pb.AllService_ListValueServer) error {
	for n := 0; n < 5; n++ {
		err := srv.Send(&pb.StreamResponse{
			StreamRes: req.Data + strconv.Itoa(n),
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
		log.Println(res.StreamReq)
	}
}

func (s *AllService) Conversations(srv pb.AllService_ConversationsServer) error {
	n := 0
	for {
		req, err := srv.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		err = srv.Send(&pb.StreamResponse{StreamRes: "from stream server answer: the " + strconv.Itoa(n) + " question is " + req.StreamReq})
		if err != nil {
			return err
		}
		n++
		log.Printf("from stream client question: %s", req.StreamReq)
	}

}

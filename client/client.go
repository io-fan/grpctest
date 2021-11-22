package main

import (
	"context"
	"io"
	"log"

	pb "grpc-example/proto"

	"google.golang.org/grpc"
)

const (
	Address string = ":8000"
)

var grpcClient pb.StreamServerClient

func main() {
	conn, err := grpc.Dial(Address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("grpc dial err:%v", err)
	}
	defer conn.Close()

	//grpcClient = pb.NewSimpleClient(conn)
	grpcClient = pb.NewStreamServerClient(conn)
	Route()
	ListValue()

}

//server stream stock
func ListValue() {
	req := pb.SimpleRequest{Data: "stream server grpc"}

	stream, err := grpcClient.ListValue(context.Background(), &req)
	if err != nil {
		log.Fatalf("call listvalue err : %v", err)
	}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("listvlalue get stream err %v", err)
		}
		log.Println(res.StreamValue)
	}
	//可以使用CloseSend()关闭stream，这样服务端就不会继续产生流消息
	//调用CloseSend()后，若继续调用Recv()，会重新激活stream，接着之前结果获取消息
	stream.CloseSend()
}

func Route() {
	req := pb.SimpleRequest{
		Data: "grpc",
	}

	res, err := grpcClient.Route(context.Background(), &req)
	if err != nil {
		log.Fatalf("call route err %v", err)
	}
	log.Println(res)
}

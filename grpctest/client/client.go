package main

import (
	"context"
	"io"
	"log"
	"strconv"

	pb "grpctest/proto"

	"google.golang.org/grpc"
)

const (
	Address string = ":8000"
)

var grpcClient pb.AllServiceClient

func main() {
	conn, err := grpc.Dial(Address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("grpc dial err:%v", err)
	}
	defer conn.Close()

	//grpcClient = pb.NewSimpleClient(conn)
	grpcClient = pb.NewAllServiceClient(conn)
	//Route()
	//ListValue()
	//RouteList()
	ConverSation()

}

func RouteList() {
	stream, err := grpcClient.RouteList(context.Background())
	if err != nil {
		log.Fatalf("upload list err %v", err)
	}
	for n := 0; n < 5; n++ {
		err := stream.Send(&pb.StreamRequest{StreamReq: "stream client rpc " + strconv.Itoa(n)})
		//发送也要检测EOF，当服务端在消息没接收完前主动调用SendAndClose()关闭stream，此时客户端还执行Send()，则会返回EOF错误，所以这里需要加上io.EOF判断
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("stream request err: %v", err)
		}
	}
	//关闭流并获取返回的消息
	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("RouteList get response err: %v", err)
	}
	log.Println(res)

}

func ConverSation() {
	stream, err := grpcClient.Conversations(context.Background())
	if err != nil {
		log.Fatalf("call conversation err:%v", err)
	}
	for n := 0; n < 500; n++ {
		err := stream.Send(&pb.StreamRequest{StreamReq: "from stream client" + strconv.Itoa(n)})
		if err != nil {
			log.Fatalf("stream request err: %v", err)
		}
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Conversations get stream err: %v", err)
		}
		log.Println(res.StreamRes)
	}
	err = stream.CloseSend()
	if err != nil {
		log.Fatalf("CloseSend err: %v", err)
	}
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
		log.Println(res.StreamRes)
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

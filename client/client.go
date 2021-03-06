package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"grpctest/client/auth"
	pb "grpctest/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

const (
	Address string = ":8000"
)

var grpcClient pb.AllServiceClient

// type PerRPCCredentials interface {
// 	GetReuqestMetadata(ctx context.Context, uri ...string) (map[string]string, error) //[]string
// 	RequireTransportSecurity() bool
// }

func main() {
	// creds, err := credentials.NewClientTLSFromFile("../pkg/tls/server_ecc.pem", "grpc-tls")
	// if err != nil {
	// 	log.Fatalf("failed to create tls credentials %v", err)
	// }
	cert, _ := tls.LoadX509KeyPair("../pkg/tls/client.pem", "../pkg/tls/client.key")
	certPool := x509.NewCertPool()
	ca, _ := ioutil.ReadFile("../pkg/tls/ca.pem")
	certPool.AppendCertsFromPEM(ca)
	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   "localhost",
		RootCAs:      certPool,
	})

	// token := auth.Token{
	// 	AppID:     "grpc_token",
	// 	AppSecret: "123456",
	// }
	token := auth.Token{
		Value: "bearer grpc.auth.token", //"bearer grpc.auth.token"
	}

	//conn, err := grpc.Dial(Address, grpc.WithInsecure())
	//conn, err := grpc.Dial(Address, grpc.WithTransportCredentials(creds))
	conn, err := grpc.Dial(Address, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(&token))
	if err != nil {
		log.Fatalf("grpc dial err:%v", err)
	}
	defer conn.Close()

	//grpcClient = pb.NewSimpleClient(conn)
	grpcClient = pb.NewAllServiceClient(conn)
	//ctx := context.Background()
	//Route(ctx, 2)
	//Route(ctx, 6)

	//ListValue()
	//RouteList()
	//ConverSation()
	routeVali()

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

func Route(ctx context.Context, deadline time.Duration) {
	clientDeadline := time.Now().Add(time.Duration(deadline * time.Second))
	ctx, cancel := context.WithDeadline(ctx, clientDeadline)
	defer cancel()

	req := pb.SimpleRequest{
		Data: "grpc",
	}

	res, err := grpcClient.Route(ctx, &req)
	if err != nil {
		state, ok := status.FromError(err)
		if ok {
			if state.Code() == codes.DeadlineExceeded {
				log.Fatalf("call route timeout")
			}

		}
		log.Fatalf("call route err %v", err)
	}
	log.Println(res)
}

func routeVali() {
	// 创建发送结构体
	req := pb.InnerMessage{
		SomeInteger: 99,
		SomeFloat:   1,
	}
	// 调用我们的服务(Route方法)
	// 同时传入了一个 context.Context ，在有需要时可以让我们改变RPC的行为，比如超时/取消一个正在运行的RPC
	res, err := grpcClient.RouteVali(context.Background(), &req)
	if err != nil {
		log.Fatalf("Call Route err: %v", err)
	}
	// 打印返回值
	log.Println(res)
}

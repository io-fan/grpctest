package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	pb "grpctest/proto"
	"io"
	"io/ioutil"
	"log"
	"net"
	"runtime"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
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
	// creds, err := credentials.NewServerTLSFromFile("../pkg/tls/server_ecc.pem", "../pkg/tls/server_ecc.key")
	// if err != nil {
	// 	log.Fatalf("failed to generate credentials %v", err)
	// }
	cert, _ := tls.LoadX509KeyPair("../pkg/tls/server.pem", "../pkg/tls/server.key")
	certPool := x509.NewCertPool()
	ca, _ := ioutil.ReadFile("../pkg/tls/ca.pem")
	certPool.AppendCertsFromPEM(ca)
	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	})
	//普通方法 一元拦截器 grpc.UnaryServerInterceptor，只拦截简单rpc，流式rpc通过grpc.StreamServerInterceptor拦截
	var interceptor grpc.UnaryServerInterceptor = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		err = Check(ctx)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
	//添加tls认证
	//grpcServer := grpc.NewServer(grpc.Creds(creds))

	//添加拦截器，增加tls认证和token认证
	grpcServer := grpc.NewServer(grpc.Creds(creds), grpc.UnaryInterceptor(interceptor))
	pb.RegisterAllServiceServer(grpcServer, &AllService{})
	log.Println(Address + " net.Listing whth TLS and token...")
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("grpcserver.serve err:%v", err)
	}
}

func Check(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx) //从上下文获取元数据
	if !ok {
		return status.Errorf(codes.Unauthenticated, "获取client token失败")
	}
	var (
		appID     string
		appSecret string
	)
	if value, ok := md["app_id"]; ok {
		appID = value[0]
	}
	if value, ok := md["app_secret"]; ok {
		appSecret = value[0]
	}
	if appID != "grpc_token" || appSecret != "123456" {
		return status.Errorf(codes.Unauthenticated, "token无效：app_id=%s,app_secret=%s", appID, appSecret)
	}
	return nil
}

func (s *AllService) Route(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {
	// token验证，添加拦截器
	// if err := Check(ctx); err != nil {
	// 	return nil, err
	// }

	//超时
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

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	pb "grpctest/proto"
	"grpctest/server/gateway"
	"grpctest/server/middleware/auth"
	"grpctest/server/middleware/recovery"
	"grpctest/server/middleware/zap"
	"io"
	"io/ioutil"
	"log"
	"net"
	"runtime"
	"strconv"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
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
	//2普通方法 一元拦截器 grpc.UnaryServerInterceptor，只拦截简单rpc，流式rpc通过grpc.StreamServerInterceptor拦截
	// var interceptor grpc.UnaryServerInterceptor = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	// 	err = Check(ctx)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	return handler(ctx, req)
	// }
	//1添加tls认证
	//grpcServer := grpc.NewServer(grpc.Creds(creds))

	//2添加拦截器，增加tls认证和token认证
	//grpcServer := grpc.NewServer(grpc.Creds(creds), grpc.UnaryInterceptor(interceptor))

	grpcServer := grpc.NewServer(grpc.Creds(creds),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			// grpc_ctxtags.StreamServerInterceptor(),
			// grpc_opentracing.StreamServerInterceptor(),
			// grpc_prometheus.StreamServerInterceptor,
			grpc_validator.StreamServerInterceptor(),
			grpc_zap.StreamServerInterceptor(zap.ZapInterceptor()),
			grpc_auth.StreamServerInterceptor(auth.AuthInterceptor),
			//grpc_recovery.StreamServerInterceptor(),
			grpc_recovery.StreamServerInterceptor(recovery.RecoveryInterceptor()),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			// grpc_ctxtags.UnaryServerInterceptor(),
			// grpc_opentracing.UnaryServerInterceptor(),
			// grpc_prometheus.UnaryServerInterceptor,
			grpc_validator.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(zap.ZapInterceptor()),
			grpc_auth.UnaryServerInterceptor(auth.AuthInterceptor),
			grpc_recovery.UnaryServerInterceptor(recovery.RecoveryInterceptor()),
		)),
	)
	pb.RegisterAllServiceServer(grpcServer, &AllService{})
	log.Println(Address + " net.Listing whth TLS and token...")
	err = grpcServer.Serve(listener)
	// if err != nil {
	// 	log.Fatalf("grpcserver.serve err:%v", err)
	// }

	//使用gateway把grpcServer转成httpServer
	httpServer := gateway.ProvideHTTP(Address, grpcServer)
	//用服务器 Serve() 方法以及我们的端口信息区实现阻塞等待，直到进程被杀死或者 Stop() 被调用
	if err = httpServer.Serve(tls.NewListener(listener, httpServer.TLSConfig)); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}

func (s *AllService) RouteVali(ctx context.Context, req *pb.InnerMessage) (*pb.OuterMessage, error) {
	// 添加拦截器后，方法里省略Token认证
	// //检测Token是否有效
	// if err := Check(ctx); err != nil {
	// 	return nil, err
	// }
	res := pb.OuterMessage{
		ImportantString: "hello grpc validator",
		Inner:           req,
	}
	return &res, nil
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

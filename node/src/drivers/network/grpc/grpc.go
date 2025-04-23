package net_grpc

import (
	"context"
	"errors"
	"fmt"
	iaction "kasper/src/abstract/models/action"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/input"
	modulelogger "kasper/src/core/module/logger"
	"kasper/src/shell/utils/future"
	"log"
	"net"
	"strings"

	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc/status"
)

type GrpcServer struct {
	app    core.ICore
	Server *grpc.Server
	logger *modulelogger.Logger
}

func ParseInput[T input.IInput](i interface{}) (input.IInput, []byte, string, error) {
	body := new(T)
	err := mapstructure.Decode(i, body)
	if err != nil {
		return nil, []byte{}, "", errors.New("invalid input format")
	}
	return *body, []byte{}, "", nil
}

func (gs *GrpcServer) serverInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	_ grpc.UnaryHandler,
) (interface{}, error) {
	fullMethod := info.FullMethod
	keyArr := strings.Split(fullMethod, "/")
	if len(keyArr) != 3 {
		return nil, status.Errorf(codes.InvalidArgument, "Wrong path format")
	}
	fmArr := strings.Split(keyArr[1], ".")
	if len(fmArr) != 2 {
		return nil, status.Errorf(codes.InvalidArgument, "Wrong path format")
	}
	key := "/" + strings.ToLower(fmArr[1][0:len(fmArr[1])-len("Service")]) + "s" + "/" + keyArr[2]
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "Metadata not provided")
	}
	userIdHeader, ok := md["userId"]
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "Authorization token is not supplied")
	}
	userId := userIdHeader[0]
	reqIdHeader, ok := md["requestId"]
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "RequestId is not supplied")
	}
	requestId := reqIdHeader[0]
	action := gs.app.Actor().FetchAction(key)
	if action == nil {
		return nil, status.Errorf(codes.NotFound, "action not found")
	}
	input, _, _, err := action.(iaction.ISecureAction).ParseInput("grpc", req)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, err.Error())
	}
	log.Println(input, requestId, userId)
	// res, _, err := action.(iaction.ISecureAction).SecurelyAct(userId, requestId, , input, "")
	// if err != nil {
		// return nil, status.Errorf(codes.Unauthenticated, err.Error())
	// } else {
	// 	return res, nil
	// }
	return nil, errors.New("not implemented")
}

func (gs *GrpcServer) Listen(port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		gs.logger.Println("failed to listen grpc: %v", err)
	}
	gs.logger.Println("server listening at %v", lis.Addr())
	future.Async(func() {
		err := gs.Server.Serve(lis)
		if err != nil {
			gs.logger.Println(err)
		}
	}, false)
}

func New(core core.ICore, logger *modulelogger.Logger) *GrpcServer {
	gs := &GrpcServer{app: core, logger: logger}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(gs.serverInterceptor),
	)
	gs.Server = grpcServer
	return gs
}

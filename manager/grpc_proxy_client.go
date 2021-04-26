package manager

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/fullstorydev/grpcurl"
	"github.com/gotomicro/ego/client/egrpc"
	"github.com/gotomicro/ego/core/elog"
	"github.com/jhump/protoreflect/grpcreflect"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

type GRPCProxyClient struct {
	name string
	addr string
	cc   *grpc.ClientConn
	desc grpcurl.DescriptorSource
	sync.RWMutex
}

func InitGRPCProxyClient(name string, addr string) *GRPCProxyClient {
	conn := egrpc.DefaultContainer().Build(
		egrpc.WithAddr(addr),
	)
	client := &GRPCProxyClient{
		cc:   conn.ClientConn,
		name: name,
		addr: addr,
	}
	err := client.initMetadata()
	if err != nil {
		elog.Panic("init metadata", elog.FieldErr(err))
	}
	return client
}

func (c *GRPCProxyClient) ProxyCall(ctx context.Context, service string, method string, data *Data, headers ...string) (*Result, error) {
	if service != "" {
		method = fmt.Sprintf("%s/%s", service, method)
	}
	md := GetMD(c.getDesc(), method)
	if md == nil {
		err := fmt.Errorf("can't get message descriptor for method %s", method)
		elog.Error("grpc call failed: ", zap.Error(err), zap.String("server", c.String()))
		result := &Result{
			Data:     nil,
			Code:     http.StatusNotFound,
			Msg:      err.Error(),
			GRPCCode: codes.NotFound,
			GRPCMsg:  err.Error(),
		}
		return result, err
	}

	body := data.GetBody()

	rf, formatter, err1 := grpcurl.RequestParserAndFormatter(grpcurl.FormatJSON, c.getDesc(), bytes.NewReader(body), grpcurl.FormatOptions{
		EmitJSONDefaultFields: true,
		IncludeTextSeparator:  true,
	})
	if err1 != nil {
		elog.Error("grpc call failed: ", zap.Error(err1), zap.String("server", c.String()))

		result := &Result{
			Data:     nil,
			Code:     http.StatusBadRequest,
			Msg:      err1.Error(),
			GRPCCode: codes.InvalidArgument,
			GRPCMsg:  err1.Error(),
		}
		return result, err1
	}
	h := NewHandler(formatter)

	err := grpcurl.InvokeRPC(ctx, c.getDesc(), c.cc, method, headers, h, rf.Next)
	if err != nil {
		elog.Error("grpc call failed: ", zap.Error(err), zap.String("server", c.String()))
		result := &Result{
			Data:     nil,
			Code:     http.StatusInternalServerError,
			Msg:      err.Error(),
			GRPCCode: codes.Unknown,
			GRPCMsg:  err.Error(),
		}
		return result, err
	}
	return h.BuildResult()
}

func (c *GRPCProxyClient) getDesc() grpcurl.DescriptorSource {
	c.RLock()
	defer c.RUnlock()
	return c.desc
}

func (c *GRPCProxyClient) String() string {
	return fmt.Sprintf("name: %s, addr: %s", c.name, c.addr)
}

func (c *GRPCProxyClient) initMetadata() error {
	pbCli := reflectpb.NewServerReflectionClient(c.cc)
	ctx := context.Background()
	md := grpcurl.MetadataFromHeaders([]string{})
	refCtx := metadata.NewOutgoingContext(ctx, md)
	refCli := grpcreflect.NewClient(refCtx, pbCli)

	if ss, err := refCli.ListServices(); err != nil {
		return err
	} else if len(ss) == 0 {
		return fmt.Errorf("server %v not support grpc reflection", c)
	}
	refCli.Reset()

	c.Lock()
	defer c.Unlock()
	c.desc = grpcurl.DescriptorSourceFromServer(ctx, refCli)
	return nil
}

func (c *GRPCProxyClient) Close() error {
	return c.cc.Close()
}

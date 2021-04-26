package manager

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/fullstorydev/grpcurl"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	gstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type grpcHandler struct {
	formatter func(proto.Message) (string, error)
	ml        sync.Mutex
	msgs      []proto.Message
	code      codes.Code
	msg       string
	details   []interface{}
}

func NewHandler(formatter func(proto.Message) (string, error)) *grpcHandler {
	return &grpcHandler{
		formatter: formatter,
	}
}

func (h *grpcHandler) BuildResult() (*Result, error) {
	code, msg := extractBizMessage(h.details)
	ret := &Result{
		Data:     make(map[string]interface{}),
		GRPCCode: h.code,
		GRPCMsg:  h.msg,
		Code:     code,
		Msg:      msg,
	}
	var err error
	var formatted string
	for _, v := range h.msgs {
		formatted, err = h.formatter(v)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(formatted), &ret.Data)
		if err != nil {
			return nil, err
		}
		break
	}
	return ret, nil
}

func convertCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.NotFound:
		return http.StatusNotFound
	case codes.Internal, codes.Unknown:
		return http.StatusInternalServerError
	case codes.DeadlineExceeded:
		return http.StatusRequestTimeout
	case codes.PermissionDenied:
		return http.StatusMethodNotAllowed
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.InvalidArgument:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// OnResolveMethod is called with a descriptor of the method that is being invoked.
func (h *grpcHandler) OnResolveMethod(*desc.MethodDescriptor) {

}

// OnSendHeaders is called with the request metadata that is being sent.
func (h *grpcHandler) OnSendHeaders(metadata.MD) {

}

// OnReceiveHeaders is called when response headers have been received.
func (h *grpcHandler) OnReceiveHeaders(md metadata.MD) {

}

func (h *grpcHandler) OnReceiveResponse(resp proto.Message) {
	h.ml.Lock()
	defer h.ml.Unlock()
	h.msgs = append(h.msgs, resp)
}

func (h *grpcHandler) OnReceiveTrailers(s *status.Status, meta metadata.MD) {
	h.code = s.Code()
	h.msg = s.Message()
	h.details = s.Details()
}

func extractBizMessage(details []interface{}) (code int, msg string) {
	for _, v := range details {
		if s, ok := v.(*gstatus.Status); ok {
			return int(s.Code), s.Message
		}
	}
	return 0, ""
}

func GetMD(ds grpcurl.DescriptorSource, method string) *desc.MessageDescriptor {
	f, _ := grpcurl.GetAllFiles(ds)
	sm := strings.Split(method, "/")
	if len(f) > 0 && len(sm) > 1 {
		var sd *desc.ServiceDescriptor
		for _, fd := range f {
			sd = fd.FindService(sm[0])
			if sd != nil {
				break
			}
		}

		if sd != nil {
			ms := sd.FindMethodByName(sm[1])
			if ms != nil {
				return ms.GetInputType()
			}
		}
	}
	return nil
}

package nrpc

import (
	"context"
	"fmt"

	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type recvMsg struct {
	ctx  context.Context
	data []byte
	req  *Request
}

func (m *recvMsg) request() (*Request, error) {
	if m.req != nil {
		return m.req, nil
	}

	var req Request
	if r := proto.Unmarshal(m.data, &req); r != nil {
		return nil, r
	}
	m.req = &req
	return m.req, nil
}

type respMsg struct {
	ctx  context.Context
	data []byte
}

func marshalProto(subj string, args proto.Message, msgType MessageType) ([]byte, error) {
	innerPayload, err := proto.Marshal(args)
	if err != nil {
		return nil, err
	}

	payload, err := proto.Marshal(&Message{
		Subject: subj,
		Data:    innerPayload,
		Type:    msgType,
	})
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func marshalReqMsg(ctx context.Context, args proto.Message, reqSubj, respSubj string, timeout int64) ([]byte, error) {
	innerPayload, err := proto.Marshal(args)
	if err != nil {
		return nil, err
	}

	md, _ := metadata.FromOutgoingContext(ctx)
	return proto.Marshal(&Request{
		Header:      fromMD(md),
		Data:        innerPayload,
		ReqSubject:  reqSubj,
		RespSubject: respSubj,
		Timeout:     timeout,
	})
}

func marshalRespMsg(resp proto.Message, header metadata.MD, trailer metadata.MD, eos bool, headerOnly bool) ([]byte, []byte, error) {
	innerPayload, err := proto.Marshal(resp)
	if err != nil {
		return nil, nil, err
	}

	payload, err := proto.Marshal(&Response{
		Header:     fromMD(header),
		Trailer:    fromMD(trailer),
		HeaderOnly: headerOnly,
		Data:       innerPayload,
		Eos:        eos,
	})
	return innerPayload, payload, err
}

func marshalUnaryRespMsg(subj string, resp proto.Message, header metadata.MD, trailer metadata.MD, eos bool, headerOnly bool) ([]byte, []byte, error) {
	innerPayload, err := proto.Marshal(resp)
	if err != nil {
		return nil, nil, err
	}

	payload, err := marshalProto(subj, &Response{
		Header:     fromMD(header),
		Trailer:    fromMD(trailer),
		HeaderOnly: headerOnly,
		Data:       innerPayload,
		Eos:        eos,
	}, MessageType_Data)
	return innerPayload, payload, err
}

func marshalEOS() ([]byte, error) {
	return proto.Marshal(&Request{
		Eos: true,
	})
}

func unmarshalReq(data []byte) (*Request, error) {
	var req Request
	if r := proto.Unmarshal(data, &req); r != nil {
		return nil, r
	}

	return &req, nil
}

func unmarshalRespMsg(data []byte, target interface{}) (*Response, error) {
	var resp Response
	if r := proto.Unmarshal(data, &resp); r != nil {
		return nil, r
	}

	// nolint: forcetypeassert
	return &resp, proto.Unmarshal(resp.GetData(), target.(proto.Message))
}

func unmarshalUnaryRespMsg(data []byte, target interface{}) (*Response, error) {
	var msg Message
	if r := proto.Unmarshal(data, &msg); r != nil {
		return nil, r
	}
	if msg.GetType() == MessageType_Error {
		return nil, unmarshalErr(msg.GetData())
	}

	var resp Response
	if r := proto.Unmarshal(msg.GetData(), &resp); r != nil {
		return nil, r
	}

	// nolint: forcetypeassert
	return &resp, proto.Unmarshal(resp.GetData(), target.(proto.Message))
}

func unmarshalErr(data []byte) error {
	var stats spb.Status
	if r := proto.Unmarshal(data, &stats); r != nil {
		return fmt.Errorf("unable to unmarshal error: %w", r)
	}
	return status.ErrorProto(&stats)
}

func toMD(header map[string]*Header) metadata.MD {
	h := metadata.MD{}
	for k, v := range header {
		h[k] = v.Values
	}
	return h
}

func fromMD(header metadata.MD) map[string]*Header {
	h := map[string]*Header{}
	for k, v := range header {
		h[k] = &Header{
			Values: v,
		}
	}
	return h
}

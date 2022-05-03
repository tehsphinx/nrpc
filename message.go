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
	resp *Response
}

func (m *respMsg) response() (*Response, error) {
	if m.resp != nil {
		return m.resp, nil
	}

	var resp Response
	if r := proto.Unmarshal(m.data, &resp); r != nil {
		return nil, r
	}
	m.resp = &resp
	return m.resp, nil
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

func unmarshalProtoReply(data []byte, target interface{}) error {
	m := &Message{}
	if r := proto.Unmarshal(data, m); r != nil {
		return r
	}

	switch m.GetType() {
	case MessageType_Data:
		return proto.Unmarshal(m.GetData(), target.(proto.Message))
	case MessageType_Error:
		var state spb.Status
		if r := proto.Unmarshal(m.GetData(), &state); r != nil {
			return r
		}
		return status.FromProto(&state).Err()
	default:
		return fmt.Errorf("unexpected message type: %v", m.GetType())
	}
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

func marshalRespMsg(resp proto.Message, header metadata.MD, eos bool) ([]byte, error) {
	innerPayload, err := proto.Marshal(resp)
	if err != nil {
		return nil, err
	}

	return proto.Marshal(&Response{
		Header: fromMD(header),
		Data:   innerPayload,
		Eos:    eos,
	})
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

func unmarshalReqMsg(data []byte, target interface{}) (*Request, error) {
	req, err := unmarshalReq(data)
	if err != nil {
		return nil, err
	}

	if r := proto.Unmarshal(data, target.(proto.Message)); r != nil {
		return nil, r
	}
	return req, nil
}

func unmarshalRespMsg(data []byte, target interface{}) (*Response, error) {
	var resp Response
	if r := proto.Unmarshal(data, &resp); r != nil {
		return nil, r
	}

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

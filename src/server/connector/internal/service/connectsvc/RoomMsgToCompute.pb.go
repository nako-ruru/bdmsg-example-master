// Code generated by protoc-gen-go. DO NOT EDIT.
// source: RoomMsgToCompute.proto

/*
Package com_mycompany_im_compute_domain is a generated protocol buffer package.

It is generated from these files:
	RoomMsgToCompute.proto

It has these top-level messages:
	FromConnectorMessage
	FromConnectorMessages
*/
package connectsvc

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type FromConnectorMessage struct {
	MessageId string            `protobuf:"bytes,1,opt,name=messageId" json:"messageId,omitempty"`
	RoomId    string            `protobuf:"bytes,2,opt,name=roomId" json:"roomId,omitempty"`
	UserId    string            `protobuf:"bytes,3,opt,name=userId" json:"userId,omitempty"`
	Nickname  string            `protobuf:"bytes,4,opt,name=nickname" json:"nickname,omitempty"`
	Time      int64             `protobuf:"varint,5,opt,name=time" json:"time,omitempty"`
	Level     int32             `protobuf:"varint,6,opt,name=level" json:"level,omitempty"`
	Type      int32             `protobuf:"varint,7,opt,name=type" json:"type,omitempty"`
	Params    map[string]string `protobuf:"bytes,8,rep,name=params" json:"params,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *FromConnectorMessage) Reset()                    { *m = FromConnectorMessage{} }
func (m *FromConnectorMessage) String() string            { return proto.CompactTextString(m) }
func (*FromConnectorMessage) ProtoMessage()               {}
func (*FromConnectorMessage) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *FromConnectorMessage) GetMessageId() string {
	if m != nil {
		return m.MessageId
	}
	return ""
}

func (m *FromConnectorMessage) GetRoomId() string {
	if m != nil {
		return m.RoomId
	}
	return ""
}

func (m *FromConnectorMessage) GetUserId() string {
	if m != nil {
		return m.UserId
	}
	return ""
}

func (m *FromConnectorMessage) GetNickname() string {
	if m != nil {
		return m.Nickname
	}
	return ""
}

func (m *FromConnectorMessage) GetTime() int64 {
	if m != nil {
		return m.Time
	}
	return 0
}

func (m *FromConnectorMessage) GetLevel() int32 {
	if m != nil {
		return m.Level
	}
	return 0
}

func (m *FromConnectorMessage) GetType() int32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *FromConnectorMessage) GetParams() map[string]string {
	if m != nil {
		return m.Params
	}
	return nil
}

type FromConnectorMessages struct {
	Messages []*FromConnectorMessage `protobuf:"bytes,1,rep,name=messages" json:"messages,omitempty"`
}

func (m *FromConnectorMessages) Reset()                    { *m = FromConnectorMessages{} }
func (m *FromConnectorMessages) String() string            { return proto.CompactTextString(m) }
func (*FromConnectorMessages) ProtoMessage()               {}
func (*FromConnectorMessages) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *FromConnectorMessages) GetMessages() []*FromConnectorMessage {
	if m != nil {
		return m.Messages
	}
	return nil
}

func init() {
	proto.RegisterType((*FromConnectorMessage)(nil), "com.mycompany.im.compute.domain.FromConnectorMessage")
	proto.RegisterType((*FromConnectorMessages)(nil), "com.mycompany.im.compute.domain.FromConnectorMessages")
}

func init() { proto.RegisterFile("RoomMsgToCompute.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 294 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x51, 0x4d, 0x4b, 0xc3, 0x40,
	0x10, 0x25, 0x4d, 0x1b, 0xdb, 0xe9, 0x45, 0x96, 0x5a, 0x96, 0x22, 0x18, 0x7a, 0xca, 0x69, 0x0f,
	0x8a, 0xa0, 0xde, 0xa4, 0x28, 0xf4, 0x50, 0xd0, 0xe0, 0xc5, 0xe3, 0x9a, 0x0c, 0x25, 0xb6, 0xb3,
	0x13, 0x76, 0xd3, 0x42, 0xfe, 0xa7, 0x3f, 0x48, 0xba, 0x59, 0xaa, 0x87, 0x82, 0xe0, 0xed, 0x7d,
	0x84, 0x97, 0x37, 0x6f, 0x61, 0x9a, 0x33, 0xd3, 0xca, 0xad, 0xdf, 0x78, 0xc1, 0x54, 0xef, 0x1a,
	0x54, 0xb5, 0xe5, 0x86, 0xc5, 0x55, 0xc1, 0xa4, 0xa8, 0x2d, 0x98, 0x6a, 0x6d, 0x5a, 0x55, 0x91,
	0x2a, 0x82, 0x5f, 0x32, 0xe9, 0xca, 0xcc, 0xbf, 0x7a, 0x30, 0x79, 0xb6, 0x4c, 0x0b, 0x36, 0x06,
	0x8b, 0x86, 0xed, 0x0a, 0x9d, 0xd3, 0x6b, 0x14, 0x97, 0x30, 0xa2, 0x0e, 0x2e, 0x4b, 0x19, 0xa5,
	0x51, 0x36, 0xca, 0x7f, 0x04, 0x31, 0x85, 0xc4, 0x32, 0xd3, 0xb2, 0x94, 0x3d, 0x6f, 0x05, 0x76,
	0xd0, 0x77, 0x0e, 0xed, 0xb2, 0x94, 0x71, 0xa7, 0x77, 0x4c, 0xcc, 0x60, 0x68, 0xaa, 0x62, 0x63,
	0x34, 0xa1, 0xec, 0x7b, 0xe7, 0xc8, 0x85, 0x80, 0x7e, 0x53, 0x11, 0xca, 0x41, 0x1a, 0x65, 0x71,
	0xee, 0xb1, 0x98, 0xc0, 0x60, 0x8b, 0x7b, 0xdc, 0xca, 0x24, 0x8d, 0xb2, 0x41, 0xde, 0x11, 0xff,
	0x65, 0x5b, 0xa3, 0x3c, 0xf3, 0xa2, 0xc7, 0xe2, 0x1d, 0x92, 0x5a, 0x5b, 0x4d, 0x4e, 0x0e, 0xd3,
	0x38, 0x1b, 0x5f, 0x3f, 0xaa, 0x3f, 0x4e, 0x56, 0xa7, 0xce, 0x55, 0x2f, 0x3e, 0xe3, 0xc9, 0x34,
	0xb6, 0xcd, 0x43, 0xe0, 0xec, 0x1e, 0xc6, 0xbf, 0x64, 0x71, 0x0e, 0xf1, 0x06, 0xdb, 0xb0, 0xc5,
	0x01, 0x1e, 0x5a, 0xee, 0xf5, 0x76, 0x87, 0x61, 0x84, 0x8e, 0x3c, 0xf4, 0xee, 0xa2, 0xf9, 0x27,
	0x5c, 0x9c, 0xfa, 0x8d, 0x13, 0xaf, 0x30, 0x0c, 0x2b, 0x3a, 0x19, 0xf9, 0xc2, 0xb7, 0xff, 0x2a,
	0x9c, 0x1f, 0x63, 0x3e, 0x12, 0xff, 0xd4, 0x37, 0xdf, 0x01, 0x00, 0x00, 0xff, 0xff, 0xfc, 0xe8,
	0x66, 0xd2, 0x04, 0x02, 0x00, 0x00,
}

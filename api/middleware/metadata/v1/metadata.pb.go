// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v4.25.3
// source: middleware/metadata/v1/metadata.proto

package v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// metadata middleware config.
type Metadata struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Prefix []string `protobuf:"bytes,1,rep,name=prefix,proto3" json:"prefix,omitempty"` // metadata prefix
}

func (x *Metadata) Reset() {
	*x = Metadata{}
	if protoimpl.UnsafeEnabled {
		mi := &file_middleware_metadata_v1_metadata_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Metadata) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Metadata) ProtoMessage() {}

func (x *Metadata) ProtoReflect() protoreflect.Message {
	mi := &file_middleware_metadata_v1_metadata_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Metadata.ProtoReflect.Descriptor instead.
func (*Metadata) Descriptor() ([]byte, []int) {
	return file_middleware_metadata_v1_metadata_proto_rawDescGZIP(), []int{0}
}

func (x *Metadata) GetPrefix() []string {
	if x != nil {
		return x.Prefix
	}
	return nil
}

var File_middleware_metadata_v1_metadata_proto protoreflect.FileDescriptor

var file_middleware_metadata_v1_metadata_proto_rawDesc = []byte{
	0x0a, 0x25, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2f, 0x6d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x6e, 0x65, 0x78, 0x74, 0x2e, 0x6d, 0x69,
	0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2e, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0x2e, 0x76, 0x31, 0x22, 0x22, 0x0a, 0x08, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61,
	0x12, 0x16, 0x0a, 0x06, 0x70, 0x72, 0x65, 0x66, 0x69, 0x78, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x06, 0x70, 0x72, 0x65, 0x66, 0x69, 0x78, 0x42, 0x36, 0x5a, 0x34, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6e, 0x65, 0x78, 0x74, 0x6d, 0x69, 0x63, 0x72, 0x6f,
	0x2f, 0x6e, 0x65, 0x78, 0x74, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65,
	0x77, 0x61, 0x72, 0x65, 0x2f, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x2f, 0x76, 0x31,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_middleware_metadata_v1_metadata_proto_rawDescOnce sync.Once
	file_middleware_metadata_v1_metadata_proto_rawDescData = file_middleware_metadata_v1_metadata_proto_rawDesc
)

func file_middleware_metadata_v1_metadata_proto_rawDescGZIP() []byte {
	file_middleware_metadata_v1_metadata_proto_rawDescOnce.Do(func() {
		file_middleware_metadata_v1_metadata_proto_rawDescData = protoimpl.X.CompressGZIP(file_middleware_metadata_v1_metadata_proto_rawDescData)
	})
	return file_middleware_metadata_v1_metadata_proto_rawDescData
}

var file_middleware_metadata_v1_metadata_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_middleware_metadata_v1_metadata_proto_goTypes = []interface{}{
	(*Metadata)(nil), // 0: next.middleware.metadata.v1.Metadata
}
var file_middleware_metadata_v1_metadata_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_middleware_metadata_v1_metadata_proto_init() }
func file_middleware_metadata_v1_metadata_proto_init() {
	if File_middleware_metadata_v1_metadata_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_middleware_metadata_v1_metadata_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Metadata); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_middleware_metadata_v1_metadata_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_middleware_metadata_v1_metadata_proto_goTypes,
		DependencyIndexes: file_middleware_metadata_v1_metadata_proto_depIdxs,
		MessageInfos:      file_middleware_metadata_v1_metadata_proto_msgTypes,
	}.Build()
	File_middleware_metadata_v1_metadata_proto = out.File
	file_middleware_metadata_v1_metadata_proto_rawDesc = nil
	file_middleware_metadata_v1_metadata_proto_goTypes = nil
	file_middleware_metadata_v1_metadata_proto_depIdxs = nil
}

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v4.25.3
// source: middleware/metrics/v1/metrics.proto

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

// Metrics middleware config.
type Metrics struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *Metrics) Reset() {
	*x = Metrics{}
	if protoimpl.UnsafeEnabled {
		mi := &file_middleware_metrics_v1_metrics_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Metrics) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Metrics) ProtoMessage() {}

func (x *Metrics) ProtoReflect() protoreflect.Message {
	mi := &file_middleware_metrics_v1_metrics_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Metrics.ProtoReflect.Descriptor instead.
func (*Metrics) Descriptor() ([]byte, []int) {
	return file_middleware_metrics_v1_metrics_proto_rawDescGZIP(), []int{0}
}

var File_middleware_metrics_v1_metrics_proto protoreflect.FileDescriptor

var file_middleware_metrics_v1_metrics_proto_rawDesc = []byte{
	0x0a, 0x23, 0x6d, 0x69, 0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2f, 0x6d, 0x65, 0x74,
	0x72, 0x69, 0x63, 0x73, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1a, 0x6e, 0x65, 0x78, 0x74, 0x2e, 0x6d, 0x69, 0x64, 0x64,
	0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x76,
	0x31, 0x22, 0x09, 0x0a, 0x07, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x42, 0x35, 0x5a, 0x33,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6e, 0x65, 0x78, 0x74, 0x6d,
	0x69, 0x63, 0x72, 0x6f, 0x2f, 0x6e, 0x65, 0x78, 0x74, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x6d, 0x69,
	0x64, 0x64, 0x6c, 0x65, 0x77, 0x61, 0x72, 0x65, 0x2f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73,
	0x2f, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_middleware_metrics_v1_metrics_proto_rawDescOnce sync.Once
	file_middleware_metrics_v1_metrics_proto_rawDescData = file_middleware_metrics_v1_metrics_proto_rawDesc
)

func file_middleware_metrics_v1_metrics_proto_rawDescGZIP() []byte {
	file_middleware_metrics_v1_metrics_proto_rawDescOnce.Do(func() {
		file_middleware_metrics_v1_metrics_proto_rawDescData = protoimpl.X.CompressGZIP(file_middleware_metrics_v1_metrics_proto_rawDescData)
	})
	return file_middleware_metrics_v1_metrics_proto_rawDescData
}

var file_middleware_metrics_v1_metrics_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_middleware_metrics_v1_metrics_proto_goTypes = []interface{}{
	(*Metrics)(nil), // 0: next.middleware.metrics.v1.Metrics
}
var file_middleware_metrics_v1_metrics_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_middleware_metrics_v1_metrics_proto_init() }
func file_middleware_metrics_v1_metrics_proto_init() {
	if File_middleware_metrics_v1_metrics_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_middleware_metrics_v1_metrics_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Metrics); i {
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
			RawDescriptor: file_middleware_metrics_v1_metrics_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_middleware_metrics_v1_metrics_proto_goTypes,
		DependencyIndexes: file_middleware_metrics_v1_metrics_proto_depIdxs,
		MessageInfos:      file_middleware_metrics_v1_metrics_proto_msgTypes,
	}.Build()
	File_middleware_metrics_v1_metrics_proto = out.File
	file_middleware_metrics_v1_metrics_proto_rawDesc = nil
	file_middleware_metrics_v1_metrics_proto_goTypes = nil
	file_middleware_metrics_v1_metrics_proto_depIdxs = nil
}

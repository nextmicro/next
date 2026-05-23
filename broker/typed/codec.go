package typed

import (
	"encoding/json"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
)

// ============================================================
// JSONCodec：内置 JSON 编解码
// ============================================================

// JSONCodec[T] 使用 encoding/json 对 T 进行编解码。
//
// 注意：T 通常应为指针类型（如 *MyStruct），以便 Unmarshal 直接写入。
type JSONCodec[T any] struct{}

// Marshal 实现 Codec[T].Marshal。
func (JSONCodec[T]) Marshal(v T) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal 实现 Codec[T].Unmarshal。
//
// 对于 T 是指针类型（如 *Foo）：需要先分配实例，再 json.Unmarshal。
// 对于 T 是值类型（如 Foo）：直接 unmarshal 到 *out。
func (JSONCodec[T]) Unmarshal(data []byte, out *T) error {
	if out == nil {
		return fmt.Errorf("typed: JSONCodec Unmarshal out is nil")
	}
	// 通过反射判断 T 是否为指针
	tt := reflect.TypeOf(out).Elem()
	if tt.Kind() == reflect.Ptr {
		// T = *X：先分配 X 的零值实例
		instance := reflect.New(tt.Elem()).Interface() // *X
		if err := json.Unmarshal(data, instance); err != nil {
			return err
		}
		// 把 *X 写回 *T（即 **X）
		v, ok := instance.(T)
		if !ok {
			return fmt.Errorf("typed: JSONCodec failed to assert to %s", tt.String())
		}
		*out = v
		return nil
	}
	// T 是值类型：直接 unmarshal 到 *out
	return json.Unmarshal(data, out)
}

// ContentType 返回 "application/json"。
func (JSONCodec[T]) ContentType() string {
	return "application/json"
}

// ============================================================
// ProtoCodec：内置 ProtoBuf 编解码
// ============================================================

// ProtoCodec[T] 使用 google.golang.org/protobuf 对实现 proto.Message 的类型进行编解码。
//
// 约束：T 必须满足 proto.Message 接口（通常为指针类型，如 *pb.MyMessage）。
//
// 用法：
//
//	var OrderCreated = typed.Topic[*pb.Order]{
//	    Name:  "orders.created",
//	    Codec: typed.ProtoCodec[*pb.Order]{},
//	}
type ProtoCodec[T proto.Message] struct{}

// Marshal 实现 Codec[T].Marshal。
func (ProtoCodec[T]) Marshal(v T) ([]byte, error) {
	return proto.Marshal(v)
}

// Unmarshal 实现 Codec[T].Unmarshal。
//
// 由于 T 是 proto.Message（指针接收者），需要通过反射创建底层结构体的新实例
// 并断言回 T，再调用 proto.Unmarshal 填充。
func (ProtoCodec[T]) Unmarshal(data []byte, out *T) error {
	if out == nil {
		return fmt.Errorf("typed: ProtoCodec Unmarshal out is nil")
	}
	tt := reflect.TypeOf(out).Elem()
	if tt.Kind() != reflect.Ptr {
		return fmt.Errorf("typed: ProtoCodec requires T to be a pointer type, got %s", tt.String())
	}
	// 创建 &Foo{} 实例
	instance := reflect.New(tt.Elem()).Interface()
	msg, ok := instance.(proto.Message)
	if !ok {
		return fmt.Errorf("typed: ProtoCodec %s does not implement proto.Message", tt.String())
	}
	if err := proto.Unmarshal(data, msg); err != nil {
		return err
	}
	// 把 proto.Message 实例断言回 T 并写入 out
	v, ok := msg.(T)
	if !ok {
		return fmt.Errorf("typed: ProtoCodec failed to assert to %s", tt.String())
	}
	*out = v
	return nil
}

// ContentType 返回 "application/x-protobuf"。
func (ProtoCodec[T]) ContentType() string {
	return "application/x-protobuf"
}

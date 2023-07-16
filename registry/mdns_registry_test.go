package registry

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-kratos/kratos/v2/registry"
)

func tcpServer(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			return
		}
		fmt.Println("get tcp")
		conn.Close()
	}
}

func TestRegistry_GetService1(t *testing.T) {
	r := NewRegistry()
	serviceInstances := []*registry.ServiceInstance{
		{
			ID:        "1",
			Name:      "server-1",
			Version:   "v0.0.1",
			Metadata:  nil,
			Endpoints: []string{"http://127.0.0.1:8000"},
		},
		{
			ID:        "2",
			Name:      "server-1",
			Version:   "v0.0.2",
			Metadata:  nil,
			Endpoints: []string{"http://127.0.0.1:8000"},
		},
	}

	for _, instance := range serviceInstances {
		err := r.Register(context.TODO(), instance)
		if err != nil {
			t.Error(err)
		}
	}

	instances, err := r.GetService(context.Background(), "server-1")
	assert.Nil(t, err)

	assert.Equal(t, len(instances), 2)
}

func TestRegistry_Register(t *testing.T) {
	type args struct {
		ctx        context.Context
		serverName string
		server     []*registry.ServiceInstance
	}

	test := []struct {
		name    string
		args    args
		want    []*registry.ServiceInstance
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				ctx:        context.Background(),
				serverName: "server-1",
				server: []*registry.ServiceInstance{
					{
						ID:        "1",
						Name:      "server-1",
						Version:   "v0.0.1",
						Metadata:  nil,
						Endpoints: []string{"http://127.0.0.1:8000"},
					},
				},
			},
			want: []*registry.ServiceInstance{
				{
					ID:      "1",
					Name:    "server-1",
					Version: "v0.0.1",
					Metadata: map[string]string{
						"kind":    "http",
						"version": "v0.0.1",
					},
					Endpoints: []string{"http://127.0.0.1:8000"},
				},
			},
			wantErr: false,
		},
		{
			name: "registry new service replace old service",
			args: args{
				ctx:        context.Background(),
				serverName: "server-1",
				server: []*registry.ServiceInstance{
					{
						ID:        "1",
						Name:      "server-1",
						Version:   "v0.0.1",
						Metadata:  nil,
						Endpoints: []string{"http://127.0.0.1:8000"},
					},
					{
						ID:        "2",
						Name:      "server-1",
						Version:   "v0.0.2",
						Metadata:  nil,
						Endpoints: []string{"http://127.0.0.1:8000"},
					},
				},
			},
			want: []*registry.ServiceInstance{
				{
					ID:      "1",
					Name:    "server-1",
					Version: "v0.0.1",
					Metadata: map[string]string{
						"kind":    "http",
						"version": "v0.0.1",
					},
					Endpoints: []string{"http://127.0.0.1:8000"},
				},
				{
					ID:      "2",
					Name:    "server-1",
					Version: "v0.0.2",
					Metadata: map[string]string{
						"kind":    "http",
						"version": "v0.0.2",
					},
					Endpoints: []string{"http://127.0.0.1:8000"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()

			for _, instance := range tt.args.server {
				err := r.Register(tt.args.ctx, instance)
				if err != nil {
					t.Error(err)
				}
			}

			watch, err := r.Watch(tt.args.ctx, tt.args.serverName)
			if err != nil {
				t.Error(err)
			}
			got, err := watch.Next()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetService() error = %v, wantErr %v", err, tt.wantErr)
				t.Errorf("GetService() got = %v", got)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetService() got = %v, want %v", got, tt.want)
			}

			for _, instance := range tt.args.server {
				_ = r.Deregister(tt.args.ctx, instance)
			}
		})
	}
}

func TestRegistry_GetService(t *testing.T) {
	addr := fmt.Sprintf("%s:9091", getIntranetIP())
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		t.Errorf("listen tcp %s failed!", addr)
		t.Fail()
	}
	defer lis.Close()
	go tcpServer(lis)
	time.Sleep(time.Millisecond * 100)
	r := NewRegistry()

	instance1 := &registry.ServiceInstance{
		ID:        "1",
		Name:      "server-1",
		Version:   "v0.0.1",
		Endpoints: []string{fmt.Sprintf("tcp://%s?isSecure=false", addr)},
	}

	instance2 := &registry.ServiceInstance{
		ID:        "2",
		Name:      "server-1",
		Version:   "v0.0.1",
		Endpoints: []string{fmt.Sprintf("tcp://%s?isSecure=false", addr)},
	}

	type fields struct {
		registry Registry
	}
	type args struct {
		ctx         context.Context
		serviceName string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      []*registry.ServiceInstance
		wantErr   bool
		preFunc   func(t *testing.T)
		deferFunc func(t *testing.T)
	}{
		{
			name:   "normal",
			fields: fields{r},
			args: args{
				ctx:         context.Background(),
				serviceName: "server-1",
			},
			want:    []*registry.ServiceInstance{instance1},
			wantErr: false,
			preFunc: func(t *testing.T) {
				if err := r.Register(context.Background(), instance1); err != nil {
					t.Error(err)
				}
				watch, err := r.Watch(context.Background(), instance1.Name)
				if err != nil {
					t.Error(err)
				}
				_, err = watch.Next()
				if err != nil {
					t.Error(err)
				}
			},
			deferFunc: func(t *testing.T) {
				err := r.Deregister(context.Background(), instance1)
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			name:   "can't get any",
			fields: fields{r},
			args: args{
				ctx:         context.Background(),
				serviceName: "server-x",
			},
			want:    nil,
			wantErr: true,
			preFunc: func(t *testing.T) {
				if err := r.Register(context.Background(), instance2); err != nil {
					t.Error(err)
				}
				watch, err := r.Watch(context.Background(), instance2.Name)
				if err != nil {
					t.Error(err)
				}
				_, err = watch.Next()
				if err != nil {
					t.Error(err)
				}
			},
			deferFunc: func(t *testing.T) {
				err := r.Deregister(context.Background(), instance2)
				if err != nil {
					t.Error(err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.preFunc != nil {
				test.preFunc(t)
			}
			if test.deferFunc != nil {
				defer test.deferFunc(t)
			}

			service, err := test.fields.registry.GetService(context.Background(), test.args.serviceName)
			if (err != nil) != test.wantErr {
				t.Errorf("GetService() error = %v, wantErr %v", err, test.wantErr)
				t.Errorf("GetService() got = %v", service)
				return
			}
			if !reflect.DeepEqual(service, test.want) {
				t.Errorf("GetService() got = %v, want %v", service, test.want)
			}
		})
	}
}

func TestRegistry_Watch(t *testing.T) {
	addr := fmt.Sprintf("%s:9091", getIntranetIP())

	time.Sleep(time.Millisecond * 100)

	instance1 := &registry.ServiceInstance{
		ID:        "1",
		Name:      "server-1",
		Version:   "v0.0.1",
		Endpoints: []string{fmt.Sprintf("tcp://%s?isSecure=false", addr)},
	}

	instance2 := &registry.ServiceInstance{
		ID:        "2",
		Name:      "server-1",
		Version:   "v0.0.1",
		Endpoints: []string{fmt.Sprintf("tcp://%s?isSecure=false", addr)},
	}

	instance3 := &registry.ServiceInstance{
		ID:        "3",
		Name:      "server-1",
		Version:   "v0.0.1",
		Endpoints: []string{fmt.Sprintf("tcp://%s?isSecure=false", addr)},
	}

	type args struct {
		ctx      context.Context
		cancel   func()
		instance *registry.ServiceInstance
	}
	canceledCtx, cancel := context.WithCancel(context.Background())

	tests := []struct {
		name    string
		args    args
		want    []*registry.ServiceInstance
		wantErr bool
		preFunc func(t *testing.T)
	}{
		{
			name: "normal",
			args: args{
				ctx:      context.Background(),
				instance: instance1,
			},
			want:    []*registry.ServiceInstance{instance1},
			wantErr: false,
			preFunc: func(t *testing.T) {
			},
		},
		{
			name: "ctx has been cancelled",
			args: args{
				ctx:      canceledCtx,
				cancel:   cancel,
				instance: instance2,
			},
			want:    nil,
			wantErr: true,
			preFunc: func(t *testing.T) {
			},
		},
		{
			name: "register with healthCheck",
			args: args{
				ctx:      context.Background(),
				instance: instance3,
			},
			want:    []*registry.ServiceInstance{instance3},
			wantErr: false,
			preFunc: func(t *testing.T) {
				lis, err := net.Listen("tcp", addr)
				if err != nil {
					t.Errorf("listen tcp %s failed!", addr)
					return
				}
				go tcpServer(lis)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preFunc != nil {
				tt.preFunc(t)
			}

			r := NewRegistry()
			err := r.Register(tt.args.ctx, tt.args.instance)
			if err != nil {
				t.Error(err)
			}
			defer func() {
				err = r.Deregister(tt.args.ctx, tt.args.instance)
				if err != nil {
					t.Error(err)
				}
			}()

			watch, err := r.Watch(tt.args.ctx, tt.args.instance.Name)
			if err != nil {
				t.Error(err)
			}

			if tt.args.cancel != nil {
				tt.args.cancel()
			}

			service, err := watch.Next()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetService() error = %v, wantErr %v", err, tt.wantErr)
				t.Errorf("GetService() got = %v", service)
				return
			}
			if !reflect.DeepEqual(service, tt.want) {
				t.Errorf("GetService() got = %v, want %v", service, tt.want)
			}
		})
	}
}

func getIntranetIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

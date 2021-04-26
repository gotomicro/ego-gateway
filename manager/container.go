package manager

import (
	"sync"

	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/core/elog"
	"go.uber.org/zap"
)

type Container struct {
	sync.RWMutex
	serviceConfigMap map[string]*Service
	componentMap     map[string]*GRPCProxyClient
}

func Init() *Container {
	container := &Container{
		componentMap: make(map[string]*GRPCProxyClient),
	}
	serviceConfigs := make([]*Service, 0)
	err := econf.UnmarshalKey("serviceConfig", &serviceConfigs)
	if err != nil {
		elog.Panic("unmarshal", elog.FieldErr(err))
	}
	serviceConfigMap := make(map[string]*Service, 0)
	for _, value := range serviceConfigs {
		serviceConfigMap[value.Name] = value
	}

	container.serviceConfigMap = serviceConfigMap
	container.updateBackend()

	// 监听
	econf.OnChange(func(config *econf.Configuration) {
		serviceConfigs := make([]*Service, 0)
		err := config.UnmarshalKey("serverConfig", &serviceConfigs)
		if err != nil {
			elog.Error("unmarshal", elog.FieldErr(err))
			return
		}
		serviceConfigMap := make(map[string]*Service, 0)
		for _, value := range serviceConfigs {
			serviceConfigMap[value.Name] = value
		}

		container.Lock()
		container.serviceConfigMap = serviceConfigMap
		container.Unlock()
		container.updateBackend()
	})
	return container
}

func (m *Container) updateBackend() {
	defer func() {
		if err := recover(); err != nil {
			elog.Error("update backend panic", zap.Any("error", err))
		}
	}()
	var deleteBackends = make([]string, 0)

	m.RLock()
	for k, b := range m.componentMap {
		if _, ok := m.serviceConfigMap[k]; !ok {
			b.Close()
			deleteBackends = append(deleteBackends, k)
		}
	}
	m.RUnlock()
	// 如果服务从配置文件中去除了，就从内存里剔除掉
	for _, k := range deleteBackends {
		m.Lock()
		delete(m.componentMap, k)
		m.Unlock()
		elog.Info("remove backend" + k)
	}

	for _, s := range m.serviceConfigMap {
		m.Lock()
		// todo 配置判断是否全部一样
		_, ok := m.componentMap[s.Name]
		if !ok {
			m.componentMap[s.Name] = InitGRPCProxyClient(s.Name, s.Addr)
		}
		m.Unlock()
	}
}

func (m *Container) GetGRPCProxyClient(name string) *GRPCProxyClient {
	m.RLock()
	defer m.RUnlock()
	return m.componentMap[name]
}

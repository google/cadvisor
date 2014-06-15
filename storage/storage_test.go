package storage

import (
	"sync"
	"testing"

	"github.com/google/cadvisor/info"
	"github.com/stretchr/testify/mock"
)

type mockStorageDriver struct {
	storageName string
	mock.Mock
}

func (self *mockStorageDriver) WriteStats(
	ref info.ContainerReference,
	stats *info.ContainerStats,
) error {
	args := self.Called(ref, stats)
	return args.Error(0)
}

type mockStorageFactory struct {
	name string
}

func (self *mockStorageFactory) String() string {
	return self.name
}

func (self *mockStorageFactory) New(
	config *Config,
) (StorageDriver, error) {
	mockWriter := &mockStorageDriver{
		storageName: self.name,
	}
	return mockWriter, nil
}

func TestContainerStatsWriterFactoryManager(t *testing.T) {
	factoryNames := []string{
		"abc",
		"bcd",
	}

	wg := sync.WaitGroup{}

	for _, name := range factoryNames {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			factory := &mockStorageFactory{
				name: n,
			}
			RegisterStorage(factory)
		}(name)
	}
	wg.Wait()
	for _, name := range factoryNames {
		wg.Add(1)
		config := &Config{
			Engine: name,
		}
		go func(n string) {
			defer wg.Done()
			writer, err := NewStorage(config)
			if err != nil {
				t.Error(err)
			}
			if mw, ok := writer.(*mockStorageDriver); ok {
				if mw.storageName != n {
					t.Errorf("wrong writer. should be %v, got %v", n, mw.storageName)
				}
			} else {
				t.Errorf("wrong writer: unknown type")
			}
		}(name)
	}
	wg.Wait()
}

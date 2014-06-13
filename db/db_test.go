package db

import (
	"sync"
	"testing"

	"github.com/google/cadvisor/info"
	"github.com/stretchr/testify/mock"
)

type mockContainerStatsWriter struct {
	dbName string
	mock.Mock
}

func (self *mockContainerStatsWriter) Write(
	ref info.ContainerReference,
	stats *info.ContainerStats,
) error {
	args := self.Called(ref, stats)
	return args.Error(0)
}

type mockContainerStatsWriterFactory struct {
	name string
}

func (self *mockContainerStatsWriterFactory) String() string {
	return self.name
}

func (self *mockContainerStatsWriterFactory) New(
	config *DatabaseConfig,
) (ContainerStatsWriter, error) {
	mockWriter := &mockContainerStatsWriter{
		dbName: self.name,
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
			factory := &mockContainerStatsWriterFactory{
				name: n,
			}
			RegisterContainerStatsWriterFactory(factory)
		}(name)
	}
	wg.Wait()
	for _, name := range factoryNames {
		wg.Add(1)
		config := &DatabaseConfig{
			Engine: name,
		}
		go func(n string) {
			defer wg.Done()
			writer, err := NewContainerStatsWriter(config)
			if err != nil {
				t.Error(err)
			}
			if mw, ok := writer.(*mockContainerStatsWriter); ok {
				if mw.dbName != n {
					t.Errorf("wrong writer. should be %v, got %v", n, mw.dbName)
				}
			} else {
				t.Errorf("wrong writer: unknown type")
			}
		}(name)
	}
	wg.Wait()
}

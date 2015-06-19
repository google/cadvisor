package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollect(t *testing.T) {
	assert := assert.New(t)

	collector, err := NewNginxCollector()
	assert.NoError(err)
	assert.Equal(collector.name, "nginx")
	
	err = SetCollectorConfig(collector, "config/sample_config.json")	
	assert.Equal(collector.configFile.Endpoint, "host:port/nginx_status")
	assert.Equal(collector.configFile.MetricsConfig[0].Name, "activeConnections")
}

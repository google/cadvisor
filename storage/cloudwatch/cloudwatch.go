package cloudwatch

import (
	"github.com/google/cadvisor/storage"
	"sync"
	"time"
	"os"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/golang/glog"
	info "github.com/google/cadvisor/info/v1"
	"flag"
	"fmt"
)

var (
	region = flag.String("storage_driver_cloudwatch_region", "us-west-1", "AWS Region")
	namespace = flag.String("storage_driver_cloudwatch_namespace", "cAdvisor",
		"AWS Cloudwatch container for metrics")
)

const (
	MaxMetricDataSize = 20 // The collection MetricData must not have a size greater than 20.
)

func init() {
	storage.RegisterStorageDriver("cloudwatch", new)
}

type cloudWatchStorage struct {
	machineName string
	namespace   string
	client      *cloudwatch.CloudWatch
	lastWrite   time.Time
	lock        sync.Mutex
}

func new() (storage.StorageDriver, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return newStorage(
		hostname,
		*region,
		*namespace,
	)
}

func (self *cloudWatchStorage) AddStats(ref info.ContainerReference, stats *info.ContainerStats) error {
	if stats == nil {
		return nil
	}

	return func() error {
		// AddStats will be invoked simultaneously from multiple threads and only one of them will perform a write.
		self.lock.Lock()
		defer self.lock.Unlock()

		metrics := self.collectMetrics(ref, stats)
		batches := self.createBatches(metrics)
		return self.sendMetrics(batches)
	}()
}

func (self *cloudWatchStorage) collectMetrics(ref info.ContainerReference, stats *info.ContainerStats) (metrics []*cloudwatch.MetricDatum) {
	extractor := metricExtractor{time.Now(), self.defaultDimensions(ref)}

	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "cpu_load_average",
		unit: cloudwatch.StandardUnitCount,
		value: float64(stats.Cpu.LoadAverage),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "cpu_usage_total",
		unit: cloudwatch.StandardUnitMicroseconds,
		value: float64(stats.Cpu.Usage.Total / uint64(time.Microsecond)),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "cpu_usage_system",
		unit: cloudwatch.StandardUnitMicroseconds,
		value: float64(stats.Cpu.Usage.System / uint64(time.Microsecond)),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "cpu_usage_user",
		unit: cloudwatch.StandardUnitMicroseconds,
		value: float64(stats.Cpu.Usage.User / uint64(time.Microsecond)),
	}))
	for i := 0; i < len(stats.Cpu.Usage.PerCpu); i++ {
		metrics = append(metrics, extractor.getDatum(metricContainer{
			metricName: "cpu_usage_per_cpu",
			unit: cloudwatch.StandardUnitMicroseconds,
			value: float64(stats.Cpu.Usage.PerCpu[i] / uint64(time.Microsecond)),
			dimensions: []*cloudwatch.Dimension{
				{Name: aws.String("cpu_core"), Value: aws.String(fmt.Sprint(i))},
			},
		}))
	}
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "memory_usage",
		unit: cloudwatch.StandardUnitBytes,
		value: float64(stats.Memory.Usage),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "memory_cache",
		unit: cloudwatch.StandardUnitBytes,
		value: float64(stats.Memory.Cache),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "memory_rss",
		unit: cloudwatch.StandardUnitBytes,
		value: float64(stats.Memory.RSS),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "memory_working_set",
		unit: cloudwatch.StandardUnitBytes,
		value: float64(stats.Memory.WorkingSet),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "memory_fail_count",
		value: float64(stats.Memory.Failcnt),
		unit: cloudwatch.StandardUnitNone,
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "nextwork_rx_bytes",
		unit: cloudwatch.StandardUnitBytes,
		value: float64(stats.Network.RxBytes),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "nextwork_rx_dropped",
		unit: cloudwatch.StandardUnitCount,
		value: float64(stats.Network.RxDropped),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "nextwork_rx_errors",
		unit: cloudwatch.StandardUnitCount,
		value: float64(stats.Network.RxErrors),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "nextwork_rx_packets",
		unit: cloudwatch.StandardUnitCount,
		value: float64(stats.Network.RxPackets),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "nextwork_tx_bytes",
		unit: cloudwatch.StandardUnitBytes,
		value: float64(stats.Network.TxBytes),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "nextwork_tx_dropped",
		unit: cloudwatch.StandardUnitCount,
		value: float64(stats.Network.TxDropped),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "nextwork_tx_errors",
		unit: cloudwatch.StandardUnitCount,
		value: float64(stats.Network.TxErrors),
	}))
	metrics = append(metrics, extractor.getDatum(metricContainer{
		metricName: "nextwork_tx_packets",
		unit: cloudwatch.StandardUnitCount,
		value: float64(stats.Network.TxPackets),
	}))
	for _, fsStat := range stats.Filesystem {
		dimensions := []*cloudwatch.Dimension{
			{Name: aws.String("device"), Value: aws.String(fsStat.Device) },
		}
		metrics = append(metrics, extractor.getDatum(metricContainer{
			metricName: "fs_usage",
			unit: cloudwatch.StandardUnitBytes,
			value: float64(fsStat.Usage),
			dimensions: append(dimensions, &cloudwatch.Dimension{
				Name: aws.String("fs_type"), Value: aws.String("usage"),
			}),
		}))
		metrics = append(metrics, extractor.getDatum(metricContainer{
			metricName: "fs_limit",
			unit: cloudwatch.StandardUnitBytes,
			value: float64(fsStat.Limit),
			dimensions: append(dimensions, &cloudwatch.Dimension{
				Name: aws.String("fs_type"), Value: aws.String("limit"),
			}),
		}))
	}

	return
}

func (self *cloudWatchStorage) createBatches(metrics []*cloudwatch.MetricDatum) (batches []*cloudwatch.PutMetricDataInput) {
	cursor := 0
	for i := range metrics {
		if (i % MaxMetricDataSize == 0 && i != 0) || i == len(metrics) - 1 {
			batches = append(batches, &cloudwatch.PutMetricDataInput{
				MetricData: metrics[i - cursor:i],
				Namespace: aws.String(self.namespace),
			})
			cursor = i
		}
	}
	return
}

func (self *cloudWatchStorage) sendMetrics(batches []*cloudwatch.PutMetricDataInput) error {
	for _, b := range batches {
		glog.V(3).Infof("Send metric data")

		glog.V(5).Infof("%#v\n", b)
		resp, err := self.client.PutMetricData(b)
		if err != nil {
			return err
		}
		glog.V(3).Infof("%#v\n", resp)
	}
	return nil
}

type metricContainer struct {
	metricName string
	unit       string
	value      float64
	dimensions []*cloudwatch.Dimension
}

type metricExtractor struct {
	time              time.Time
	defaultDimensions []*cloudwatch.Dimension
}

func (self *metricExtractor) getDatum(metric metricContainer) *cloudwatch.MetricDatum {
	return &cloudwatch.MetricDatum{
		MetricName: aws.String(metric.metricName),
		Dimensions: append(self.defaultDimensions, metric.dimensions...),
		Timestamp: aws.Time(self.time),
		Unit:      aws.String(metric.unit),
		Value:     aws.Float64(metric.value),
	}
}

func (self *cloudWatchStorage) defaultDimensions(ref info.ContainerReference) []*cloudwatch.Dimension {
	dimensions := []*cloudwatch.Dimension{
		{
			Name: aws.String("Hostname"),
			Value: aws.String(self.machineName),
		},
		{
			Name: aws.String("ContainerName"),
			Value: aws.String(ref.Name),
		},
	}
	if ref.Id != "" {
		dimensions = append(dimensions, &cloudwatch.Dimension{
			Name: aws.String("ContainerId"),
			Value: aws.String(ref.Id),
		})
	}
	for i, l := range ref.Labels {
		dimensions = append(dimensions, &cloudwatch.Dimension{
			Name: aws.String(fmt.Sprintf("label%v", i)),
			Value: aws.String(l),
		})
	}
	return dimensions
}

func (self *cloudWatchStorage) Close() error {
	self.client = nil
	return nil
}

func newStorage(machineName string, region string, namespace string) (*cloudWatchStorage, error) {
	svc := cloudwatch.New(session.New(), &aws.Config{Region: aws.String(region)})

	if err := testConnection(svc, namespace); err != nil {
		glog.Errorf("Cloudwatch: cannot establish connection – %s", err.Error())
		return &cloudWatchStorage{}, err
	} else {
		glog.Infoln("cloudwatch storage initialized")
	}

	return &cloudWatchStorage{
		machineName: machineName,
		namespace: namespace,
		lastWrite: time.Now(),
		client: svc,
	}, nil
}

func testConnection(svc *cloudwatch.CloudWatch, namespace string) (err error) {
	params := &cloudwatch.ListMetricsInput{
		Namespace: aws.String(namespace),
	}
	_, err = svc.ListMetrics(params)
	return
}
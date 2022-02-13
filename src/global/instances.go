package global

import (
	"github.com/viderstv/api/src/monitoring/prometheus"
	"github.com/viderstv/common/instance"
)

type Instances struct {
	Redis      instance.Redis
	Mongo      instance.Mongo
	RMQ        instance.RabbitMQ
	Prometheus prometheus.Instance
}

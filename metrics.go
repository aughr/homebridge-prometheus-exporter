package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type HomebridgeCollector struct {
	session *Session
	config  *Config
}

func NewHomebridgeCollector(session *Session, config *Config) *HomebridgeCollector {
	return &HomebridgeCollector{
		session: session,
		config:  config,
	}
}

func (c *HomebridgeCollector) Describe(ch chan<- *prometheus.Desc) {
	// We cannot know what metrics we will have until we collect them
	// so we send no descriptors - this is a valid approach for dynamic metrics
}

func (c *HomebridgeCollector) Collect(ch chan<- prometheus.Metric) {
	token, err := c.session.GetToken()
	if err != nil {
		log.Printf("Error getting token: %v", err)
		return
	}

	accessories, err := getAllAccessories(token, c.config.URI, c.config.Debug)
	if err != nil {
		log.Printf("Error fetching accessories: %v", err)
		return
	}

	for _, accessory := range accessories {
		for _, service := range accessory.ServiceCharacteristics {
			if strings.EqualFold(service.Format, "string") {
				continue
			}

			value, err := convertToFloat64(service.Value)
			if err != nil {
				if c.config.Debug {
					log.Printf("Skipping service %s: %v", service.ServiceName, err)
				}
				continue
			}

			metricName := fmt.Sprintf("%s_%s_%s",
				c.config.Prefix,
				toSnakeCase(service.ServiceType),
				toSnakeCase(service.Type),
			)

			desc := prometheus.NewDesc(
				metricName,
				service.Description,
				[]string{"name"},
				nil,
			)

			metric := prometheus.MustNewConstMetric(
				desc,
				prometheus.GaugeValue,
				value,
				toSnakeCase(service.ServiceName),
			)

			ch <- metric
		}
	}
}

func convertToFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, nil
		}
		return 0, fmt.Errorf("cannot convert string '%s' to float64", v)
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

func toSnakeCase(s string) string {
	var result strings.Builder

	for i, r := range s {
		if i > 0 && (r >= 'A' && r <= 'Z') {
			result.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			result.WriteRune(r + 32)
		} else if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		} else {
			result.WriteByte('_')
		}
	}

	s = result.String()
	s = strings.Trim(s, "_")

	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}

	return s
}

package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

func buildRegistry(token, uri, prefix string, debug bool) (*prometheus.Registry, error) {
	registry := prometheus.NewRegistry()

	accessories, err := getAllAccessories(token, uri, debug)
	if err != nil {
		log.Printf("Error fetching accessories: %v", err)
		return nil, err
	}

	for _, accessory := range accessories {
		for _, service := range accessory.ServiceCharacteristics {
			if strings.EqualFold(service.Format, "string") {
				continue
			}

			value, err := convertToFloat64(service.Value)
			if err != nil {
				if debug {
					log.Printf("Skipping service %s: %v", service.ServiceName, err)
				}
				continue
			}

			metricName := fmt.Sprintf("%s_%s_%s",
				prefix,
				toSnakeCase(service.ServiceType),
				toSnakeCase(service.Type),
			)

			gauge := prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: metricName,
					Help: service.Description,
				},
				[]string{"name"},
			)

			if err := registry.Register(gauge); err != nil {
				if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
					gauge = are.ExistingCollector.(*prometheus.GaugeVec)
				} else {
					log.Printf("Failed to register metric %s: %v", metricName, err)
					continue
				}
			}

			gauge.WithLabelValues(toSnakeCase(service.ServiceName)).Set(value)
		}
	}

	return registry, nil
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

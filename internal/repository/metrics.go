// Package repository работает с хранением метрик.
//
// Разделен на две части.
// На хранение метрик во временном хранилище.
// На хранение в физическом хранилище (БД и/или файл).
// Метрики хранятся в двух типах.
package repository

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/cryptohash"
	"github.com/andynikk/advancedmetrics/internal/encoding"
)

// Gauge тип метрики хранения на базе float64
type Gauge float64

// Counter тип метрики хранения на базе float64
type Counter int64

type MutexRepo map[string]Metric

type MapMetrics struct {
	MutexRepo
}

type Metric interface {
	String() string
	Type() string
	Set(v encoding.Metrics)
	SetFromText(metValue string) bool
	GetMetrics(id string, mType string, hashKey string) encoding.Metrics
}

// String возаращает значение метрики строкой
func (g *Gauge) String() string {

	return fmt.Sprintf("%g", *g)
}

// Type возаращает тип значения метрики строкой
func (g *Gauge) Type() string {
	return "gauge"
}

// GetMetrics Сохраняет метрику в формате encoding.Metrics.
// И возращает ее в вызываемую процедуру.
func (g *Gauge) GetMetrics(mType string, id string, hashKey string) encoding.Metrics {

	value := float64(*g)
	msg := fmt.Sprintf("%s:%s:%f", id, mType, value)
	heshVal := cryptohash.HeshSHA256(msg, hashKey)

	mt := encoding.Metrics{ID: id, MType: mType, Value: &value, Hash: heshVal}

	return mt
}

// Set Устанавливает значение метрики в тип Gauge из типа float64
func (g *Gauge) Set(v encoding.Metrics) {

	*g = Gauge(*v.Value)

}

// SetFromText Устанавливает значение метрики в тип Gauge из типа string
func (g *Gauge) SetFromText(metValue string) bool {

	predVal, err := strconv.ParseFloat(metValue, 64)
	if err != nil {
		constants.Logger.ErrorLog(errors.New("error convert type"))

		return false
	}
	*g = Gauge(predVal)

	return true

}

///////////////////////////////////////////////////////////////////////////////

// Set Устанавливает значение метрики в тип Counter из типа int64
func (c *Counter) Set(v encoding.Metrics) {

	*c = *c + Counter(*v.Delta)
}

// SetFromText Устанавливает значение метрики в тип Counter из типа string
func (c *Counter) SetFromText(metValue string) bool {

	predVal, err := strconv.ParseInt(metValue, 10, 64)
	if err != nil {
		constants.Logger.ErrorLog(errors.New("error convert type"))

		return false
	}
	*c = *c + Counter(predVal)

	return true

}

// String возаращает значение метрики строкой
func (c *Counter) String() string {

	return fmt.Sprintf("%d", *c)
}

// GetMetrics Сохраняет метрику в формате encoding.Metrics.
// И возращает ее в вызываемую процедуру.
func (c *Counter) GetMetrics(mType string, id string, hashKey string) encoding.Metrics {

	delta := int64(*c)

	msg := fmt.Sprintf("%s:%s:%d", id, mType, delta)
	heshVal := cryptohash.HeshSHA256(msg, hashKey)

	mt := encoding.Metrics{ID: id, MType: mType, Delta: &delta, Hash: heshVal}

	return mt
}

// Type возаращает тип значения метрики строкой
func (c *Counter) Type() string {
	return "counter"
}

////////////////////////////////////////////////////////////////////////////////

func (mm *MapMetrics) TextMetricsAndValue() []string {
	const msgFormat = "%s = %s"

	var msg []string

	for key, val := range mm.MutexRepo {
		msg = append(msg, fmt.Sprintf(msgFormat, key, val.String()))
	}

	return msg
}

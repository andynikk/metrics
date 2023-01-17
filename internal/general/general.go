package general

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/andynikk/advancedmetrics/internal/compression"
	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/constants/errs"
	"github.com/andynikk/advancedmetrics/internal/cryptohash"
	"github.com/andynikk/advancedmetrics/internal/encoding"
	"github.com/andynikk/advancedmetrics/internal/encryption"
	"github.com/andynikk/advancedmetrics/internal/environment"
	"github.com/andynikk/advancedmetrics/internal/grpchandlers"
	"github.com/andynikk/advancedmetrics/internal/handlers/api"
	"github.com/andynikk/advancedmetrics/internal/repository"
)

type MetricType int

const (
	GaugeMetric MetricType = iota
	CounterMetric
)

func (mt MetricType) String() string {
	return [...]string{"gauge", "counter"}[mt]
}

type IRepStore interface {
	api.RepStore | grpchandlers.RepStore
}

type Header map[string]string

type RepStore[T IRepStore] struct {
	data map[string]T
}

func New[T IRepStore]() RepStore[T] {

	c := RepStore[T]{}
	c.data = make(map[string]T)

	return c
}

func (rs *RepStore[T]) Set(key string, value T) {
	rs.data[key] = value
}

func (rs *RepStore[T]) Get(key string) (v T) {
	if v, ok := rs.data[key]; ok {
		return v
	}

	return
}

func (rs *RepStore[T]) getPKRepStore() *encryption.KeyEncryption {
	if t, ok := rs.data[constants.TypeSrvGRPC.String()]; ok {
		return any(t).(grpchandlers.RepStore).PK
	}

	if t, ok := rs.data[constants.TypeSrvHTTP.String()]; ok {
		return any(t).(api.RepStore).PK
	}

	return &encryption.KeyEncryption{}
}

func (rs *RepStore[T]) getConfigRepStore() *environment.ServerConfig {
	if t, ok := rs.data[constants.TypeSrvGRPC.String()]; ok {
		return any(t).(grpchandlers.RepStore).Config
	}

	if t, ok := rs.data[constants.TypeSrvHTTP.String()]; ok {
		return any(t).(api.RepStore).Config
	}

	return &environment.ServerConfig{}
}

func (rs *RepStore[T]) getSyncMapMetricsRepStore() *repository.SyncMapMetrics {
	if t, ok := rs.data[constants.TypeSrvGRPC.String()]; ok {
		return any(t).(grpchandlers.RepStore).SyncMapMetrics
	}

	if t, ok := rs.data[constants.TypeSrvHTTP.String()]; ok {
		return any(t).(api.RepStore).SyncMapMetrics
	}

	return &repository.SyncMapMetrics{}
}

func (rs *RepStore[T]) RestoreData() {

	var arrMetricsAll []encoding.Metrics

	config := rs.getConfigRepStore()
	storageType := config.StorageType

	for _, valMetric := range storageType {

		arrMetrics, err := valMetric.GetMetric()
		if err != nil {
			constants.Logger.ErrorLog(err)
			continue
		}
		arrMetricsAll = append(arrMetricsAll, arrMetrics...)
	}

	rs.SetValueInMapJSON(arrMetricsAll)
}

func (rs *RepStore[T]) SetValueInMapJSON(a []encoding.Metrics) error {

	key := rs.getConfigRepStore().Key
	smm := rs.getSyncMapMetricsRepStore()

	smm.Lock()
	defer smm.Unlock()

	for _, v := range a {
		var heshVal string

		switch v.MType {
		case GaugeMetric.String():
			var valValue float64
			valValue = *v.Value

			msg := fmt.Sprintf("%s:gauge:%f", v.ID, valValue)
			heshVal = cryptohash.HeshSHA256(msg, key)
			if _, findKey := smm.MutexRepo[v.ID]; !findKey {
				valG := repository.Gauge(0)
				smm.MutexRepo[v.ID] = &valG
			}
		case CounterMetric.String():
			var valDelta int64
			valDelta = *v.Delta

			msg := fmt.Sprintf("%s:counter:%d", v.ID, valDelta)
			heshVal = cryptohash.HeshSHA256(msg, key)
			if _, findKey := smm.MutexRepo[v.ID]; !findKey {
				valC := repository.Counter(0)
				smm.MutexRepo[v.ID] = &valC
			}
		default:
			return errs.ErrNotImplemented
		}

		heshAgent := []byte(v.Hash)
		heshServer := []byte(heshVal)

		hmacEqual := hmac.Equal(heshServer, heshAgent)

		if v.Hash != "" && !hmacEqual {
			constants.Logger.InfoLog(fmt.Sprintf("++ %s - %s", v.Hash, heshVal))
			return errs.ErrBadRequest
		}
		smm.MutexRepo[v.ID].Set(v)
	}
	return nil

}

func (rs *RepStore[T]) BackupData() {

	rsConfig := rs.getConfigRepStore()
	storageType := rsConfig.StorageType
	storeInterval := rsConfig.StoreInterval

	ctx, cancelFunc := context.WithCancel(context.Background())
	saveTicker := time.NewTicker(storeInterval)
	for {
		select {
		case <-saveTicker.C:

			for _, val := range storageType {
				val.WriteMetric(rs.PrepareDataForBackup())
			}

		case <-ctx.Done():
			cancelFunc()
			return
		}
	}
}

func (rs *RepStore[T]) PrepareDataForBackup() encoding.ArrMetrics {

	cKey := rs.getConfigRepStore().Key
	smm := rs.getSyncMapMetricsRepStore()

	var storedData encoding.ArrMetrics
	for key, val := range smm.MutexRepo {
		storedData = append(storedData, val.GetMetrics(val.Type(), key, cKey))
	}
	return storedData
}

// Добавляет в хранилище метрику. Определяет тип метрики (gauge, counter).
// В зависимости от типа добавляет нужное значение.
// При успешном выполнении возвращает http-статус "ОК" (200)
func (rs *RepStore[T]) setValueInMap(metType string, metName string, metValue string) int {

	smm := rs.getSyncMapMetricsRepStore()
	switch metType {
	case GaugeMetric.String():
		if val, findKey := smm.MutexRepo[metName]; findKey {
			if ok := val.SetFromText(metValue); !ok {
				return http.StatusBadRequest
			}
		} else {

			valG := repository.Gauge(0)
			if ok := valG.SetFromText(metValue); !ok {
				return http.StatusBadRequest
			}

			smm.MutexRepo[metName] = &valG
		}

	case CounterMetric.String():
		if val, findKey := smm.MutexRepo[metName]; findKey {
			if ok := val.SetFromText(metValue); !ok {
				return http.StatusBadRequest
			}
		} else {

			valC := repository.Counter(0)
			if ok := valC.SetFromText(metValue); !ok {
				return http.StatusBadRequest
			}

			smm.MutexRepo[metName] = &valC
		}
	default:
		return http.StatusNotImplemented
	}

	return http.StatusOK
}

// Shutdown working out the service stop.
// We save the current values of metrics in the database.
func (rs *RepStore[T]) Shutdown() {

	storageType := rs.getConfigRepStore().StorageType
	smm := rs.getSyncMapMetricsRepStore()

	smm.Lock()
	defer smm.Unlock()

	for _, val := range storageType {
		val.WriteMetric(rs.PrepareDataForBackup())
	}
	constants.Logger.InfoLog("server stopped")
}

// HandlerUpdatesMetricJSON Handler, который работает с POST запросом формата "/updates".
// В теле получает массив JSON-значений со значением метрики. Струтура JSON: encoding.Metrics.
// Может принимать JSON в жатом виде gzip. Сохраняет значение в физическое и временное хранилище.
func (rs *RepStore[T]) HandlerUpdatesMetricJSON(h Header, b []byte) error {

	contentEncoding := h["content-encoding"]
	contentEncryption := h["content-encryption"]

	PK := rs.getPKRepStore()

	err := errors.New("")
	if strings.Contains(contentEncryption, constants.TypeEncryption) {
		b, err = PK.RsaDecrypt(b)

		if err != nil {
			constants.Logger.ErrorLog(err)
			return errs.ErrDecrypt
		}
	}

	if strings.Contains(contentEncoding, "gzip") {
		_, err := compression.Decompress(b)
		if err != nil {
			constants.Logger.ErrorLog(err)
			return errs.ErrDecompress
		}
	}
	if err := rs.Updates(b); err != nil {
		return err
	}

	return nil
}

// HandlerUpdateMetricJSON Handler, который работает с POST запросом формата "/update".
// В теле получает JSON со значением метрики. Струтура JSON: encoding.Metrics.
// Может принимать JSON в жатом виде gzip.
// Сохраняет значение в физическое и временное хранилище.
func (rs *RepStore[T]) HandlerUpdateMetricJSON(h Header, b []byte) error {

	contentEncoding := h["content-encoding"]
	contentEncryption := h["content-encryption"]

	err := errors.New("")
	if strings.Contains(contentEncryption, constants.TypeEncryption) {
		PK := rs.getPKRepStore()
		b, err = PK.RsaDecrypt(b)
		if err != nil {
			constants.Logger.ErrorLog(err)
			return errs.ErrDecrypt
		}
	}

	if strings.Contains(contentEncoding, "gzip") {
		b, err = compression.Decompress(b)
		if err != nil {
			constants.Logger.InfoLog(fmt.Sprintf("$$ 2 %s", err.Error()))
			return errs.ErrDecompress
		}
	}

	bodyJSON := bytes.NewReader(b)

	var v []encoding.Metrics
	err = json.NewDecoder(bodyJSON).Decode(&v)
	if err != nil {
		constants.Logger.InfoLog(fmt.Sprintf("$$ 3 %s", err.Error()))
		return errs.ErrGetJSON
	}

	err = rs.SetValueInMapJSON(v)
	if err != nil {
		constants.Logger.ErrorLog(err)
		return err
	}
	smm := rs.getSyncMapMetricsRepStore()
	cfg := rs.getConfigRepStore()

	var arrMetrics encoding.ArrMetrics
	for _, val := range v {
		mt := smm.MutexRepo[val.ID].GetMetrics(val.MType, val.ID, cfg.Key)
		arrMetrics = append(arrMetrics, mt)
	}

	for _, val := range cfg.StorageType {
		val.WriteMetric(arrMetrics)
	}

	return nil
}

func (rs *RepStore[T]) HandlerSetMetricaPOST(metType string, metName string, metValue string) error {

	smm := rs.getSyncMapMetricsRepStore()
	smm.Lock()
	defer smm.Unlock()

	res := rs.setValueInMap(metType, metName, metValue)
	switch res {
	case 200:
		return nil
	case 400:
		return errs.ErrBadRequest
	case 501:
		return errs.ErrNotImplemented
	default:
		return errs.ErrStatusInternalServer
	}
}

// HandlerPingDB Handler, который работает с GET запросом формата "/ping"
// Handler проверяет соединение с физическим хранилищем метрик.
// Физическое хранилище регулируется параметром среды "DATABASE_DSN" или флагом "d"
// Если заполнено "DATABASE_DSN" или "d", то это база данных. Иначе файл.
func (rs *RepStore[T]) HandlerPingDB(h Header) error {

	cfg := rs.getConfigRepStore()

	mapTypeStore := cfg.StorageType
	if _, findKey := mapTypeStore[constants.MetricsStorageDB.String()]; !findKey {
		constants.Logger.ErrorLog(errors.New("соединение с базой отсутствует"))
		return errs.ErrStatusInternalServer
	}

	if mapTypeStore[constants.MetricsStorageDB.String()].ConnDB() == nil {
		constants.Logger.ErrorLog(errors.New("соединение с базой отсутствует"))
		return errs.ErrStatusInternalServer
	}
	return nil
}

func (rs *RepStore[T]) Updates(msg []byte) error {

	bodyJSON := bytes.NewReader(msg)
	respByte, err := io.ReadAll(bodyJSON)

	if err != nil {
		constants.Logger.ErrorLog(err)
		return errs.ErrStatusInternalServer
	}

	var storedData encoding.ArrMetrics
	if err = json.Unmarshal(respByte, &storedData); err != nil {
		constants.Logger.ErrorLog(err)
		return errs.ErrStatusInternalServer
	}

	if err = rs.SetValueInMapJSON(storedData); err != nil {
		return err
	}

	storageType := rs.getConfigRepStore().StorageType
	for _, val := range storageType {
		val.WriteMetric(storedData)
	}

	return nil
}

// HandlerValueMetricaJSON Handler, который работает с POST запросом формата "/value".
// В теле получает JSON с имененм типа и именем метрики. Струтура JSON: encoding.Metrics.
// Может принимать JSON в жатом виде gzip. Возвращает значение метрики по типу и наименованию.
func (rs *RepStore[T]) HandlerValueMetricaJSON(h Header, b []byte) (Header, []byte, error) {

	acceptEncoding := h["accept-encoding"]
	contentEncoding := h["content-encoding"]
	contentEncryption := h["content-encryption"]

	PK := rs.getPKRepStore()
	err := errors.New("")
	if strings.Contains(contentEncryption, constants.TypeEncryption) {
		b, err = PK.RsaDecrypt(b)
		if err != nil {
			constants.Logger.ErrorLog(err)
			return nil, nil, errs.ErrDecrypt
		}
	}

	if strings.Contains(contentEncoding, "gzip") {
		b, err = compression.Decompress(b)
		if err != nil {
			constants.Logger.ErrorLog(err)
			return nil, nil, errs.ErrDecompress
		}
	}

	bodyJSON := bytes.NewReader(b)

	v := encoding.Metrics{}
	err = json.NewDecoder(bodyJSON).Decode(&v)
	if err != nil {
		constants.Logger.ErrorLog(err)
		return nil, nil, errs.ErrGetJSON
	}
	metType := v.MType
	metName := v.ID

	smm := rs.getSyncMapMetricsRepStore()
	cfg := rs.getConfigRepStore()

	smm.Lock()
	defer smm.Unlock()

	if _, findKey := smm.MutexRepo[metName]; !findKey {

		constants.Logger.InfoLog(fmt.Sprintf("== %d %s %d %s", 1, metName, len(smm.MutexRepo), cfg.DatabaseDsn))
		return nil, nil, errs.ErrNotFound
	}

	mt := smm.MutexRepo[metName].GetMetrics(metType, metName, cfg.Key)
	metricsJSON, err := mt.MarshalMetrica()
	if err != nil {
		constants.Logger.ErrorLog(err)
		return nil, nil, err
	}

	var bytMterica []byte
	bt := bytes.NewBuffer(metricsJSON).Bytes()
	bytMterica = append(bytMterica, bt...)
	compData, err := compression.Compress(bytMterica)
	if err != nil {
		constants.Logger.ErrorLog(err)
	}

	hReturn := Header{}
	hReturn["content-type"] = "application/json"

	var bodyBate []byte
	if strings.Contains(acceptEncoding, "gzip") {
		hReturn["content-encoding"] = "gzip"
		bodyBate = compData
	} else {
		bodyBate = metricsJSON
	}

	return hReturn, bodyBate, nil
}

// HandlerGetValue Handler, который работает с GET запросом формата "/value/{metType}/{metName}"
// Где metType наименование типа метрики, metName наименование метрики
func (rs *RepStore[T]) HandlerGetValue(metName []byte) (string, error) {

	smm := rs.getSyncMapMetricsRepStore()

	smm.Lock()
	defer smm.Unlock()

	if _, findKey := smm.MutexRepo[string(metName)]; !findKey {
		constants.Logger.InfoLog(fmt.Sprintf("== %d", 3))
		return "", errs.ErrStatusInternalServer
	}

	strMetric := smm.MutexRepo[string(metName)].String()
	return strMetric, nil

}

// HandlerGetAllMetrics Отрабатывает обращение к корневому узлу сервера (/).
// Выводит на страницу список наименований и значений метрик.
func (rs *RepStore[T]) HandlerGetAllMetrics(h Header) (Header, []byte) {

	smm := rs.getSyncMapMetricsRepStore()
	arrMetricsAndValue := smm.MapMetrics.TextMetricsAndValue()

	var strMetrics string
	content := `<!DOCTYPE html>
				<html>
				<head>
  					<meta charset="UTF-8">
  					<title>МЕТРИКИ</title>
				</head>
				<body>
				<h1>МЕТРИКИ</h1>
				<ul>
				`
	for _, val := range arrMetricsAndValue {
		content = content + `<li><b>` + val + `</b></li>` + "\n"
		if strMetrics != "" {
			strMetrics = strMetrics + ";"
		}
		strMetrics = strMetrics + val
	}
	content = content + `</ul>
						</body>
						</html>`

	acceptEncoding := h["Accept-Encoding"]

	metricsHTML := []byte(content)
	byteMterics := bytes.NewBuffer(metricsHTML).Bytes()
	compData, err := compression.Compress(byteMterics)
	if err != nil {
		constants.Logger.ErrorLog(err)
	}

	HeaderResponse := Header{}

	var bodyBate []byte
	if strings.Contains(acceptEncoding, "gzip") {
		HeaderResponse["content-encoding"] = "gzip"
		bodyBate = compData
	} else {
		bodyBate = metricsHTML
	}

	HeaderResponse["content-type"] = "text/html"
	HeaderResponse["metrics-val"] = strMetrics

	return HeaderResponse, bodyBate
}

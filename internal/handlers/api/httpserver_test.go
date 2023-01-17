package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/encryption"
	"github.com/andynikk/advancedmetrics/internal/environment"
	"github.com/andynikk/advancedmetrics/internal/general"
	"github.com/andynikk/advancedmetrics/internal/repository"
)

var srv HTTPServer

func (s *HTTPServer) ExampleRepStore_HandlerGetAllMetrics() {
	ts := httptest.NewServer(srv.Router)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/", strings.NewReader(""))
	if err != nil {
		return
	}
	defer req.Body.Close()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	msg := fmt.Sprintf("Metrics: %s. HTTP-Status: %d",
		resp.Header.Get("Metrics-Val"), resp.StatusCode)
	fmt.Println(msg)
	_ = resp.Close

	// Output:
	// Metrics: TestGauge = 0.001. HTTP-Status: 200
}

func (s *HTTPServer) ExampleRepStore_HandlerSetMetricaPOST() {

	ts := httptest.NewServer(srv.Router)
	defer ts.Close()

	req, err := http.NewRequest("POST", ts.URL+"/update/gauge/TestGauge/0.01", strings.NewReader(""))
	if err != nil {
		return
	}
	defer req.Body.Close()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	fmt.Print(resp.StatusCode)

	// Output:
	// 200
}

func (s *HTTPServer) ExampleRepStore_HandlerGetValue() {

	ts := httptest.NewServer(srv.Router)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/value/gauge/TestGauge", strings.NewReader(""))
	if err != nil {
		return
	}
	defer req.Body.Close()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	fmt.Print(resp.StatusCode)

	// Output:
	// 200
}

func init() {

	storage := RepStore{}

	// NewRepStore инициализация хранилища, роутера, заполнение настроек.

	smm := new(repository.SyncMapMetrics)
	smm.MutexRepo = make(repository.MutexRepo)
	storage.SyncMapMetrics = smm

	sc := environment.ServerConfig{}
	sc.InitConfigServerENV()
	sc.InitConfigServerFile()
	sc.InitConfigServerDefault()

	storage.Config = &sc

	storage.PK, _ = encryption.InitPrivateKey(storage.Config.CryptoKey)

	storage.Config.StorageType, _ = repository.InitStoreDB(storage.Config.StorageType, storage.Config.DatabaseDsn)
	storage.Config.StorageType, _ = repository.InitStoreFile(storage.Config.StorageType, storage.Config.StoreFile)

	gRepStore := general.New[RepStore]()
	gRepStore.Set(constants.TypeSrvHTTP.String(), storage)

	srv.RepStore = gRepStore

	rp := srv.RepStore.Get(constants.TypeSrvHTTP.String())
	rp.MutexRepo = make(repository.MutexRepo)
	srv.InitRoutersMux()

	valG := repository.Gauge(0)
	if ok := valG.SetFromText("0.001"); !ok {
		return
	}
	rp.MutexRepo["TestGauge"] = &valG
}

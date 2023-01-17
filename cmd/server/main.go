// Start of the service for getting metrics.
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/encryption"
	"github.com/andynikk/advancedmetrics/internal/environment"
	"github.com/andynikk/advancedmetrics/internal/general"
	"github.com/andynikk/advancedmetrics/internal/grpchandlers"
	. "github.com/andynikk/advancedmetrics/internal/grpchandlers/api"
	"github.com/andynikk/advancedmetrics/internal/handlers/api"
	"github.com/andynikk/advancedmetrics/internal/middlware"
)

type serverHTTP struct {
	storage api.RepStore
	*api.HTTPServer
}

type serverGRPS struct {
	storage grpchandlers.RepStore
	srv     GRPCServer
}

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

type Server interface {
	Start() error
	RestoreData()
	BackupData()
	Shutdown()
}

func (s *serverHTTP) Start() error {
	HTTPServer := &http.Server{
		Addr:    s.storage.Config.Address,
		Handler: s.Router,
	}

	if err := HTTPServer.ListenAndServe(); err != nil {
		return err
	}

	return nil
}

func (s *serverGRPS) Start() error {

	server := grpc.NewServer(middlware.WithServerUnaryInterceptor())
	RegisterMetricCollectorServer(server, &s.srv)
	l, err := net.Listen("tcp", constants.AddressServer)
	if err != nil {
		return err
	}

	if err = server.Serve(l); err != nil {
		return err
	}

	return nil
}

func (s *serverHTTP) RestoreData() {
	if s.storage.Config.Restore {
		s.RepStore.RestoreData()
	}
}

func (s *serverGRPS) RestoreData() {
	if s.storage.Config.Restore {
		s.srv.RepStore.RestoreData()
	}
}

func (s *serverHTTP) BackupData() {
	s.RepStore.BackupData()
}

func (s *serverGRPS) BackupData() {
	s.srv.RepStore.BackupData()
}

func (s *serverHTTP) Shutdown() {
	s.RepStore.Shutdown()
}

func (s *serverGRPS) Shutdown() {
	s.srv.RepStore.Shutdown()
}

func newHTTPServer(configServer *environment.ServerConfig) *serverHTTP {

	server := new(serverHTTP)

	server.storage.Config = configServer
	server.storage.PK, _ = encryption.InitPrivateKey(configServer.CryptoKey)
	api.NewRepStore(&server.storage)
	fmt.Println(&server.storage.Config.Address)

	gRepStore := general.New[api.RepStore]()
	gRepStore.Set(constants.TypeSrvHTTP.String(), server.storage)
	server.HTTPServer = &api.HTTPServer{
		RepStore: gRepStore,
	}
	server.HTTPServer.InitRoutersMux()
	return server
}

func newGRPCServer(configServer *environment.ServerConfig) *serverGRPS {
	server := new(serverGRPS)

	server.storage.Config = configServer
	server.storage.PK, _ = encryption.InitPrivateKey(configServer.CryptoKey)

	grpchandlers.NewRepStore(&server.storage)
	fmt.Println(server.storage.Config.Address)

	gRepStore := general.New[grpchandlers.RepStore]()
	gRepStore.Set(constants.TypeSrvGRPC.String(), server.storage)

	srv := &GRPCServer{
		RepStore: gRepStore,
	}
	server.srv = *srv

	return server
}

// NewServer реализует фабричный метод.
func NewServer(configServer *environment.ServerConfig) Server {
	if configServer.TypeServer == constants.TypeSrvGRPC.String() {
		return newGRPCServer(configServer)
	}

	return newHTTPServer(configServer)
}

func main() {

	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	config := environment.InitConfigServer()

	server := NewServer(config)
	go server.RestoreData()
	go server.BackupData()

	gRepStore := general.New[api.RepStore]()
	//gRepStore.Set(constants.TypeSrvHTTP.String(), server)
	srv := &api.HTTPServer{
		RepStore: gRepStore,
	}
	srv.InitRoutersMux()

	go func() {
		HTTPServer := &http.Server{
			Addr:    config.Address,
			Handler: srv.Router,
		}

		if err := HTTPServer.ListenAndServe(); err != nil {
			constants.Logger.ErrorLog(err)
			return
		}

		//err := server.Start()
		//if err != nil {
		//	constants.Logger.ErrorLog(err)
		//	return
		//}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	<-stop
	server.Shutdown()
}

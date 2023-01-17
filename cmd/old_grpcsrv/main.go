package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/general"
	"github.com/andynikk/advancedmetrics/internal/grpchandlers"
	"github.com/andynikk/advancedmetrics/internal/grpchandlers/api"
	"github.com/andynikk/advancedmetrics/internal/middlware"
	"google.golang.org/grpc"
)

type server struct {
	storage grpchandlers.RepStore
}

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {

	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	server := new(server)
	grpchandlers.NewRepStore(&server.storage)
	fmt.Println(server.storage.Config.Address)

	gRepStore := general.New[grpchandlers.RepStore]()
	gRepStore.Set(constants.TypeSrvGRPC.String(), server.storage)

	if server.storage.Config.Restore {
		go gRepStore.RestoreData()
	}

	go gRepStore.BackupData()

	s := grpc.NewServer(middlware.WithServerUnaryInterceptor())
	srv := &api.GRPCServer{
		RepStore: gRepStore,
	}

	api.RegisterMetricCollectorServer(s, srv)
	l, err := net.Listen("tcp", constants.AddressServer)
	if err != nil {
		log.Fatal(err)
	}

	go func() {

		if err = s.Serve(l); err != nil {
			log.Fatal(err)
		}

	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	<-stop
	gRepStore.Shutdown()
}

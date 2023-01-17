package grpchandlers

import (
	"github.com/andynikk/advancedmetrics/internal/encryption"
	"github.com/andynikk/advancedmetrics/internal/environment"
	"github.com/andynikk/advancedmetrics/internal/repository"
)

type RepStore struct {
	Config *environment.ServerConfig
	PK     *encryption.KeyEncryption
	//Router *mux.Router
	*repository.SyncMapMetrics
}

// NewRepStore инициализация хранилища, роутера, заполнение настроек.
func NewRepStore(rs *RepStore) {

	smm := new(repository.SyncMapMetrics)
	smm.MutexRepo = make(repository.MutexRepo)
	rs.SyncMapMetrics = smm

	InitRoutersMux(rs)

	//rs.Config = environment.InitConfigGRPC()
	//rs.Config = environment.InitConfigServer()
	//rs.PK, _ = encryption.InitPrivateKey(rs.Config.CryptoKey)
	//
	//rs.Config.StorageType, _ = repository.InitStoreDB(rs.Config.StorageType, rs.Config.DatabaseDsn)
	//rs.Config.StorageType, _ = repository.InitStoreFile(rs.Config.StorageType, rs.Config.StoreFile)
}

// InitRoutersMux создание роутера.
// Описание методов для обработки handlers сервера
func InitRoutersMux(rs *RepStore) {

	//s := grpc.NewServer()
	//srv := &api.GRPCServer{}
	//api.RegisterUpdatersServer(s, srv)
	//l, err := net.Listen("tcp", constants.AddressServer)
	//if err != nil {
	//	log.Fatal(err)
	//}
}

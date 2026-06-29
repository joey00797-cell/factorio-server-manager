package factorio

import (
	"log"
	"strconv"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"

	"github.com/OpenFactorioServerManager/rcon"
)

func connectRC() error {
	return GetFactorioServer().connectRC()
}

func (server *Server) connectRC() error {
	var err error
	config := bootstrap.GetConfig()
	host := "127.0.0.1"
	if config.ServerIP != "" && config.ServerIP != "0.0.0.0" {
		host = config.ServerIP
	}
	rconAddr := host + ":" + strconv.Itoa(server.rconPort())
	server.Rcon, err = rcon.Dial(rconAddr, server.rconPass())
	if err != nil {
		log.Printf("Cannot create rcon session: %s", err)
		return err
	}
	log.Printf("rcon session established on %s", rconAddr)

	return nil
}

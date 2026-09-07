package factorio

import (
	"log"
	"strconv"

	"github.com/OpenFactorioServerManager/rcon"
)

func connectRC() error {
	return GetFactorioServer().connectRC()
}

func (server *Server) connectRC() error {
	var err error
	host := "127.0.0.1"
	// Always connect to localhost - ServerIP is for display only
	rconAddr := host + ":" + strconv.Itoa(server.rconPort())
	server.Rcon, err = rcon.Dial(rconAddr, server.rconPass())
	if err != nil {
		log.Printf("Cannot create rcon session: %s", err)
		return err
	}
	log.Printf("rcon session established on %s", rconAddr)

	return nil
}

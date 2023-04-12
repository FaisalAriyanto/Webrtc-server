package turn

import (
	"fmt"
	"net"
	"strconv"

	"github.com/pion/turn/v2"
)

type TurnServerConfig struct {
	PublicIP string
	Port     int
	Realm    string
}

func DefaultConfig() TurnServerConfig {
	return TurnServerConfig{
		PublicIP: "127.0.0.1",
		Port:     19302,
		Realm:    "flutter-webrtc",
	}
}

type TurnServer struct {
	udpListener net.PacketConn
	turnServer  *turn.Server
	Config      TurnServerConfig
	AuthHandler func(username string, realm string, srcAddr net.Addr) (string, bool)
}

func authHandler(username string, realm string, srcAddr net.Addr) (string, bool) {
	if username == "faisal" {
		return "123456", true
	}
	return "", false
}

func NewTurnServer(config TurnServerConfig) *TurnServer {

	server := &TurnServer{
		Config:      config,
		AuthHandler: authHandler,
	}
	if len(config.PublicIP) == 0 {
		fmt.Print("'public-ip' is required")
	}
	udpListener, err := net.ListenPacket("udp4", "0.0.0.0:"+strconv.Itoa(config.Port))
	if err != nil {
		fmt.Printf("Failed to create TURN server listener: %s", err)
	}
	server.udpListener = udpListener

	turnServer, err := turn.NewServer(turn.ServerConfig{
		Realm:       config.Realm,
		AuthHandler: server.HandleAuthenticate,
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(config.PublicIP),
					Address:      "0.0.0.0",
				},
			},
		},
	})
	if err != nil {
		fmt.Printf("%v", err)
	}
	server.turnServer = turnServer

	fmt.Printf("TURN server started at %s:%d\n", config.PublicIP, config.Port)
	return server
}

func (s *TurnServer) HandleAuthenticate(username string, realm string, srcAddr net.Addr) ([]byte, bool) {
	if s.AuthHandler != nil {
		if password, ok := s.AuthHandler(username, realm, srcAddr); ok {
			fmt.Printf("Authentication succeeded for user '%s' with IP '%s'\n", username, srcAddr.String())
			return turn.GenerateAuthKey(username, realm, password), true
		}
	}
	fmt.Printf("Authentication failed for user '%s' with IP '%s'\n", username, srcAddr.String())
	return nil, false
}

func (s *TurnServer) Close() error {
	return s.turnServer.Close()
}

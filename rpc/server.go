package rpc

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gocoin/blockchain"
)

type Server struct {
	Port int
	gin  *gin.Engine
}

func NewServer(port int, bc *blockchain.Blockchain) *Server {
	return &Server{
		Port: port,
		gin:  NewRouter(bc),
	}
}

func (s *Server) Run() error {
	err := s.gin.Run()

	if err != nil {
		return fmt.Errorf("cannot start API Server: %w", err)
	}

	return nil
}

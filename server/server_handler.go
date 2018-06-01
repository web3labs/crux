package server

import (
	"golang.org/x/net/context"
)

type Server struct {
}

func (s *Server) Version(ctx context.Context, in *Tm) (*ApiVersion, error) {
	return &ApiVersion{Version:apiVersion}, nil
}

func (s *Server) Upcheck(ctx context.Context, in *Tm) (*UpCheckResp, error) {
	return &UpCheckResp{Upcheck:upCheckResponse}, nil
}
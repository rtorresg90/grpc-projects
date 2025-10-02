package main

import(
    "log"
    "net"
    "sync"

    "google.golang.org/grpc"

    pb "github.com/rtorresg90/grpc-projects/chat/proto"
)

var addr string = "localhost:1990"

type client struct {
    user string
    send chan *pb.ServerEvent
}

type Hub struct {
    mu    sync.RWMutex
    rooms map[int32]map[string]*client // roomID -> user -> client
}

type Server struct {
    pb.UnimplementedChatServiceServer
    hub *Hub
}

func main() {

    listen, err := net.Listen("tcp", addr)
    if err != nil {
        log.Fatalf("Failed to listen on %v\n", err)
    }

    opts := []grpc.ServerOption{}
    // Add TLS creds here

    s := grpc.NewServer(opts...)
    pb.RegisterChatServiceServer(s, &Server{hub: initializeHub()})
    if err = s.Serve(listen); err != nil {
        log.Fatalf("Failed to serve: %v\n")
    }
}

func initializeHub() *Hub {
    return &Hub{
        rooms: make(map[int32]map[string]*client),
    }
}
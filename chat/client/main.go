package main

import(
    "flag"
    "log"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    pb "github.com/rtorresg90/grpc-projects/chat/proto"
)

func main() {

   addr := flag.String("addr", "localhost:1990", "gRPC server address")
   room := flag.Int("room", 1, "room id")
   user := flag.String("user", "", "username (required)")
   flag.Parse()
   if *user == "" {
        log.Fatal("missing -user")
    }

	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials())) // no tls for now
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
    defer conn.Close()

    client := pb.NewChatServiceClient(conn)
    err = doChat(client, int32(*room), *user)
    if err != nil {
        log.Fatalf("chat: %v", err)
    }
}
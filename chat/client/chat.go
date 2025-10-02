package main

import(
    "bufio"
    "context"
    "fmt"
    "io"
    "os"
    "log"
    "strings"

    pb "github.com/rtorresg90/grpc-projects/chat/proto"
)

func doChat(client pb.ChatServiceClient, room int32, user string) error {
    stream, err := client.Chat(context.Background())
    if err != nil {
        return err
    }

    joinEvent := &pb.ClientEvent{
              		Payload: &pb.ClientEvent_Join {
              			Join: &pb.Join{
              				Room: room,
              				User: user,
              			},
              		},
              	}

    err = stream.Send(joinEvent)
    if err != nil {
        return err
    }

    done := make(chan struct{})
    go func() {
        defer close(done)
        for {
            event, err := stream.Recv()
            if err != nil {
                if err != io.EOF {
                    log.Printf("Receive error: %v\n", err)
                }
                return
            }
            switch payload := event.Payload.(type) {
                case *pb.ServerEvent_Notifications:
                    n := payload.Notifications
                    fmt.Printf("[room=%d][system]: %s\n", n.GetRoom(), n.GetMessage())
                case *pb.ServerEvent_Text:
                    t := payload.Text
                    fmt.Printf("[room=%d][%s]: %s\n", t.GetRoom(), t.GetUser(), t.GetMessage())
                default:
                    log.Printf("Unknown payload %v\n", payload)
            }
        }
    }()

    log.Printf("Connected as %q in room%d. Type /quit to exit.")
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            continue
        }

        if line == "/quit" {
            _ = stream.CloseSend()
            <-done
            return nil
        }

        textEvent := &pb.ClientEvent{
                        Payload: &pb.ClientEvent_Text {
                            Text: &pb.Text{
                                Room: room,
                                User: user,
                                Message: line,
                                },
                            },
                        }

        err = stream.Send(textEvent)
        if err != nil {
            log.Printf("Send error %v\n", err)
            _ = stream.CloseSend()
            <-done
            return nil
        }
    }

    _ = stream.CloseSend()
    <-done

    return nil
}
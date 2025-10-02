package main

import(
    "errors"
    "fmt"
    "io"
    "log"

    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    pb "github.com/rtorresg90/grpc-projects/chat/proto"
)


func (s *Server) Chat(stream pb.ChatService_ChatServer) error {
        // get initial request and check for errors or client disconnect
        req, err := stream.Recv()
        if err != nil {
            if errors.Is(err, io.EOF) {
                return nil
            }
            return err
        }

        // initial request must be a Join.
        j := req.GetJoin()
        if j == nil || j.User == "" {
            _ = stream.Send(&pb.ServerEvent{
                Payload: &pb.ServerEvent_Notifications{
                    Notifications: &pb.Notifications{Room: -1, Message: "join (room, user) required"},
                },
            })

            return status.Error(codes.InvalidArgument, "join required")
        }

        room, user := j.Room, j.User
        // initialize a client channel
        cl := &client{
            user: user,
            send: make(chan *pb.ServerEvent, 32),
        }

        // register user in hub
        if err := s.hub.add(room, user, cl); err != nil {
            return status.Error(codes.AlreadyExists, "user is already registered")
        }

        // Announce user joined
        event := &pb.ServerEvent{
                    Payload: &pb.ServerEvent_Notifications{
                        Notifications: &pb.Notifications{Room: room, Message: user + " joined the room"},
                    },
                }
        s.hub.broadcast(room, event)

        // each client will have its own goroutine that will send messages to the stream.
        go func() {
            for event := range cl.send {
                if err := stream.Send(event); err != nil {
                    return
                }
            }
        }()

        for {
            req, err := stream.Recv()
            if err != nil {
                break // client exited
            }

            switch p := req.Payload.(type) {
                case *pb.ClientEvent_Join:
                    log.Printf("[join] Ignoring, user %s already belongs to room %d", user, room)
                case *pb.ClientEvent_Text:
                    _ = s.hub.text(room, user, p.Text.Message)
                //case *pb.ClientEvent_Typing:
                //    ...

                default:
                    log.Printf("Unknown message type, ignoring")
            }
        }

        if removedClient, _ := s.hub.remove(room, user); removedClient != nil {
            close(removedClient.send)
        }

        event = &pb.ServerEvent{
            Payload: &pb.ServerEvent_Notifications{
                Notifications: &pb.Notifications{Room: room, Message: user + " left the room"},
            },
        }

        s.hub.broadcast(room, event)
        return nil
    }

func (h *Hub) add(room int32, user string, cl *client) error {
    log.Printf("[join] user=%s room=%d\n", user, room)
    h.mu.Lock()
    defer h.mu.Unlock()
    if h.rooms[room] == nil {
        h.rooms[room] = make(map[string]*client)
    }
    if _, exists := h.rooms[room][user]; exists {
        return fmt.Errorf("[join] user %q already joined room %d", user, room)
    }

    h.rooms[room][user] = cl
    return nil
}

func (h *Hub) text(room int32, fromUser string, message string) error {
    log.Printf("[text] user=%s room=%d\n", fromUser, room)
    if message == "" {
        return nil
    }
    h.mu.RLock()
	roomMap, ok := h.rooms[room]
    if !ok || len(roomMap) == 0 {
        h.mu.RUnlock()
        return fmt.Errorf("[text] room %d doesn't exist", room)
    }
    if _, exists := h.rooms[room][fromUser]; !exists {
        h.mu.RUnlock()
        return fmt.Errorf("[text] user %q is not assigned to room %d", fromUser, room)
    }
    h.mu.RUnlock()

    event := &pb.ServerEvent{
        Payload: &pb.ServerEvent_Text{
    	    Text: &pb.Text{
    	        Room:    room,
    		    User:    fromUser,
    		    Message: message,
    	    },
        },
    }

    h.broadcast(room, event)

    return nil

}

//func (h *Hub) typing(room int32, user string) error {
//    return nil
//}



func (h *Hub) remove(room int32, user string) (*client, error) {
    h.mu.Lock()
    defer h.mu.Unlock()
    log.Printf("[remove] user=%s room=%d\n", user, room)

    roomMap, ok := h.rooms[room]
    if !ok {
        return nil, fmt.Errorf("[remove] room %d doesn't exist", room)
    }

	cl, exists := roomMap[user]
	if !exists {
		return nil, fmt.Errorf("[remove] user %q is not assigned to room %d", user, room)
	}

    delete(roomMap, user)
    if len(roomMap) == 0 {
        delete(h.rooms, room)
    }
    return cl, nil
}


func (h *Hub) broadcast(room int32, evt *pb.ServerEvent) {
        h.mu.RLock()
        roomMap := h.rooms[room]
        recipients := getRecipients(roomMap)
        h.mu.RUnlock()

        doBroadcast(evt, recipients)
}

func doBroadcast(evt *pb.ServerEvent, recipients []*client) {
	dropped := 0
	for _, rcpt := range recipients {
		select {
		case rcpt.send <- evt:
		default:
			dropped++ // means channel is full
		}
	}
	if dropped > 0 {
		log.Printf("broadcastText: dropped %d messages", dropped)
	}
}

func getRecipients(roomMap map[string]*client) []*client {
    recipients := make([]*client, 0, len(roomMap))
    for _, c := range roomMap {
        recipients = append(recipients, c)
    }

    return recipients
}



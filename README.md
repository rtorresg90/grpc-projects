# grpc-projects
GRPC Chat application.

Users can join different chat rooms and exchange messages with other users who are registered in the same room.
	•	Each user establishes a gRPC stream with the server.
	•	The server manages active rooms and users through a central Hub.
	•	Messages are broadcast only to members of the same room.
	•	When a user disconnects, the server cleans up and notifies others in the room.

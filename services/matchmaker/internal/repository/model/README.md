# model

## The lifecycle of models

### QueuedPlayer

A QueuedPlayer will exist from the time a ticket is created until the ticket is completely finished with.
This means it will be present until a Match is created (or a PendingMatch becomes a Match).

### PendingMatch

A PendingMatch exists for gamemodes that don't have instant matchmaking. A PendingMatch will exist
until the countdown is finished, and it is converted into a Match. A PendingMatch may also be deleted
if the countdown is cancelled (e.g. due to configuration changes) or if the minimum gamemode requirements are
no longer satisfied (e.g. the minimum player count is no longer satisfied).

### Match

A Match model doesn't exist as it isn't stored in the database. It exists only as a protobuf.
A Match is created when an instant match occurs or a PendingMatch is converted into one.
When a Match is created, it is at this point a server is allocated and a message is fired.

### Ticket

A Ticket represents one or more players that are queueing for gamemode. It may also contain data on the party
that player(s) belong to. A Ticket is created when a player queues for a gamemode and is deleted when a Match is made.

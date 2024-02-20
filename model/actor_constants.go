package model

// ActorAnnounceChildren signifies that an actor/Stream should broadcast changes to its child Streams.
const ActorAnnounceChildren = "CHILDREN"

// ActorAnnounceChildReplies signifies that an actor/Stream should broadcast all replies/mentions to its children.
const ActorAnnounceChildReplies = "CHILD-REPLIES"

// ActorAnnounceReplies signifies that an actor/Stream should re-broadcast all replies/mentions to itself.
const ActorAnnounceReplies = "REPLIES"

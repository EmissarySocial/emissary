package writer

import "github.com/benpate/activitystream/vocabulary"

// Accept
func Accept(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeAccept).
		Actor(actor).
		Object(object)
}

// Add
func Add(actor Object, object Object, target Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeAdd).
		Actor(actor).
		Object(object).
		Target(target)
}

// Announce
func Announce(actor Object, object Object, target Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeAnnounce).
		Actor(actor).
		Object(object).
		Target(target)
}

// Arrive
func Arrive(actor Object, location Object, origin Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeArrive).
		Actor(actor).
		Location(location).
		Origin(origin)
}

// Block
func Block(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeBlock).
		Actor(actor).
		Object(object)
}

// Create
func Create(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeCreate).
		Actor(actor).
		Object(object)
}

// Delete
func Delete(actor Object, object Object, origin Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeDelete).
		Actor(actor).
		Object(object).
		Origin(origin)
}

// Dislike
func Dislike(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeDislike).
		Actor(actor).
		Object(object)
}

// Flag
func Flag(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeFlag).
		Actor(actor).
		Object(object)
}

// Follow
func Follow(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeFollow).
		Actor(actor).
		Object(object)
}

// Ignore
func Ignore(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeIgnore).
		Actor(actor).
		Object(object)
}

// Invite
func Invite(actor Object, object Object, target Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeInvite).
		Actor(actor).
		Object(object).
		Target(target)
}

// Join
func Join(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeJoin).
		Actor(actor).
		Object(object)
}

// Leave
func Leave(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeLeave).
		Actor(actor).
		Object(object)
}

// Like
func Like(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeLike).
		Actor(actor).
		Object(object)
}

// Listen
func Listen(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeListen).
		Actor(actor).
		Object(object)
}

// Move
func Move(actor Object, object Object, origin Object, target Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeMove).
		Actor(actor).
		Object(object).
		Origin(origin).
		Target(target)
}

// Offer
func Offer(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeLike).
		Actor(actor).
		Object(object)
}

// Question
func Question() Object {
	// TODO: this is not complete
	return NewObject().
		Type(vocabulary.ActivityTypeQuestion)
}

// Reject
func Reject(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeReject).
		Actor(actor).
		Object(object)
}

// Read
func Read(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeRead).
		Actor(actor).
		Object(object)
}

// Remove
func Remove(actor Object, object Object, origin Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeRemove).
		Actor(actor).
		Object(object).
		Origin(origin)
}

// TentativeAccept
func TentativeAccept(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeTentativeAccept).
		Actor(actor).
		Object(object)
}

// TentativeReject
func TentativeReject(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeTentativeReject).
		Actor(actor).
		Object(object)
}

// Travel
func Travel(actor Object, origin Object, target Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeTravel).
		Actor(actor).
		Origin(origin).
		Target(target)
}

// Undo
func Undo(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeUndo).
		Actor(actor).
		Object(object)
}

// Update
func Update(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeUpdate).
		Actor(actor).
		Object(object)
}

// View
func View(actor Object, object Object) Object {
	return NewObject().
		Type(vocabulary.ActivityTypeView).
		Actor(actor).
		Object(object)
}

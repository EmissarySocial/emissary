package model

import (
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestCollectionSchema confirms that every Collection field round-trips through its JSON-Schema
func TestCollectionSchema(t *testing.T) {

	collection := NewCollection()
	s := schema.New(CollectionSchema())

	table := []tableTestItem{
		{"collectionId", "123456781234567812345678", nil},
		{"userId", "aaa4bbb8ddd4ddd812345678", nil},
		{"parentId", "bbb4ccc8eee4eee912345678", nil},
		{"parentType", "Stream", nil},
		{"collectionType", "Context", nil},
		{"read.0", "https://johnconnor.mil/@john", nil},
		{"read.1", "https://sarah.sky.net/@sarah", nil},
		{"write.0", "https://kyle.mil/@reese", nil},
	}

	tableTest_Schema(t, &s, &collection, table)
}

// TestNewCollection confirms that a fresh Collection is fully initialized
func TestNewCollection(t *testing.T) {

	collection := NewCollection()

	// A fresh Collection has a generated (non-zero) CollectionID
	require.False(t, collection.CollectionID.IsZero())

	// Read/Write slices are initialized (non-nil) and empty
	require.NotNil(t, collection.Read)
	require.NotNil(t, collection.Write)
	require.Zero(t, len(collection.Read))
	require.Zero(t, len(collection.Write))

	// The remaining ObjectIDs default to zero
	require.True(t, collection.UserID.IsZero())
	require.True(t, collection.ParentID.IsZero())

	// The string "type" fields default to empty
	require.Empty(t, collection.ParentType)
	require.Empty(t, collection.CollectionType)
}

// TestCollection_ID confirms that ID returns the hex encoding of the CollectionID
func TestCollection_ID(t *testing.T) {

	collection := NewCollection()

	// ID() returns the hex encoding of the CollectionID
	require.Equal(t, collection.CollectionID.Hex(), collection.ID())
}

// TestCollection_Fields checks the projection against the struct's actual bson tags. The previous
// version of this test simply restated the literal, so it passed for as long as the list named
// four fields ("collectionId", "to", "cc", "name") that Collection has never had.
func TestCollection_Fields(t *testing.T) {

	collection := NewCollection()

	require.Equal(t, []string{"_id", "userId", "parentId", "parentType", "collectionType", "read", "write", "totalItems"}, collection.Fields())
	require.Subset(t, bsonNames(Collection{}), collection.Fields(), "every projected name must be a real bson field")
}

/******************************************
 * AccessLister Interface
 ******************************************/

// TestCollection_State confirms that a Collection reports a single, constant workflow state
func TestCollection_State(t *testing.T) {

	collection := NewCollection()

	require.Equal(t, "DEFAULT", collection.State())
}

// TestCollection_IsAuthor confirms that only the owning User is the author, and the zero UserID never is
func TestCollection_IsAuthor(t *testing.T) {

	userID := primitive.NewObjectID()
	otherID := primitive.NewObjectID()

	collection := NewCollection()
	collection.UserID = userID

	// The owner is the author
	require.True(t, collection.IsAuthor(userID))

	// A different user is not the author
	require.False(t, collection.IsAuthor(otherID))

	// A zero UserID is never the author, even when the collection's UserID is also zero.
	empty := NewCollection()
	require.True(t, empty.UserID.IsZero())
	require.False(t, empty.IsAuthor(primitive.NilObjectID))
}

// TestCollection_IsMyself confirms that a Collection never represents a User directly
func TestCollection_IsMyself(t *testing.T) {

	collection := NewCollection()

	// A Collection never directly represents a User, so IsMyself is always FALSE.
	require.False(t, collection.IsMyself(primitive.NewObjectID()))
	require.False(t, collection.IsMyself(primitive.NilObjectID))
}

// TestCollection_RolesToGroupIDs confirms that roles map onto the default Group permissions
func TestCollection_RolesToGroupIDs(t *testing.T) {

	userID := primitive.NewObjectID()

	collection := NewCollection()
	collection.UserID = userID

	// The "author" role maps to the owner's UserID; "anonymous" maps to the magic anonymous group.
	result := collection.RolesToGroupIDs(MagicRoleAuthor, MagicRoleAnonymous)

	require.Equal(t, Permissions{userID, MagicGroupIDAnonymous}, result)

	// No roles yields an empty (but non-nil) Permissions slice.
	require.Zero(t, len(collection.RolesToGroupIDs()))
}

// TestCollection_RolesToPrivilegeIDs confirms that a Collection grants no Privileges
func TestCollection_RolesToPrivilegeIDs(t *testing.T) {

	collection := NewCollection()
	collection.UserID = primitive.NewObjectID()

	// A Collection grants no privileges via roles, regardless of the roles requested.
	require.Equal(t, NewPermissions(), collection.RolesToPrivilegeIDs(MagicRoleAuthor, MagicRoleAnonymous))
	require.Equal(t, NewPermissions(), collection.RolesToPrivilegeIDs())
}

/******************************************
 * Read / Write Permissions
 ******************************************/

// TestCollection_Readable confirms that only actors on the Read list may read a Collection
func TestCollection_Readable(t *testing.T) {

	const alice = "https://alice.test/@alice"
	const bob = "https://bob.test/@bob"

	collection := NewCollection()
	collection.Read = sliceof.String{alice}

	// An actor named in the Read list can read; NotReadable is its inverse.
	require.True(t, collection.IsReadable(alice))
	require.False(t, collection.NotReadable(alice))

	// An actor not named in the Read list cannot read.
	require.False(t, collection.IsReadable(bob))
	require.True(t, collection.NotReadable(bob))
}

// TestCollection_Readable_Public confirms that the Public namespace on the Read list opens a Collection to everyone
func TestCollection_Readable_Public(t *testing.T) {

	const stranger = "https://stranger.test/@nobody"

	// The public namespace token in the Read list makes the collection readable by
	// any actor, including one not named in the list and an empty actor.
	collection := NewCollection()
	collection.Read = sliceof.String{vocab.NamespacePublic}

	require.True(t, collection.IsReadable(stranger))
	require.True(t, collection.IsReadable(""))
	require.False(t, collection.NotReadable(stranger))
}

// TestCollection_Readable_Empty confirms that an empty Read list denies everyone
func TestCollection_Readable_Empty(t *testing.T) {

	// A Collection with an empty Read list is readable by no one.
	collection := NewCollection()

	require.False(t, collection.IsReadable("https://anyone.test/@x"))
	require.True(t, collection.NotReadable("https://anyone.test/@x"))
	require.False(t, collection.IsReadable(""))
}

// TestCollection_Writable confirms that only actors on the Write list may write to a Collection
func TestCollection_Writable(t *testing.T) {

	const alice = "https://alice.test/@alice"
	const bob = "https://bob.test/@bob"

	collection := NewCollection()
	collection.Write = sliceof.String{alice}

	// An actor named in the Write list can write; NotWritable is its inverse.
	require.True(t, collection.IsWritable(alice))
	require.False(t, collection.NotWritable(alice))

	// An actor not named in the Write list cannot write.
	require.False(t, collection.IsWritable(bob))
	require.True(t, collection.NotWritable(bob))
}

// TestCollection_Writable_Public confirms that the Public namespace on the Write list opens a Collection to everyone
func TestCollection_Writable_Public(t *testing.T) {

	const stranger = "https://stranger.test/@nobody"

	// The public namespace token in the Write list makes the collection writable by
	// any actor, including one not named in the list and an empty actor.
	collection := NewCollection()
	collection.Write = sliceof.String{vocab.NamespacePublic}

	require.True(t, collection.IsWritable(stranger))
	require.True(t, collection.IsWritable(""))
	require.False(t, collection.NotWritable(stranger))
}

// TestCollection_Writable_Empty confirms that an empty Write list denies everyone
func TestCollection_Writable_Empty(t *testing.T) {

	// A Collection with an empty Write list is writable by no one.
	collection := NewCollection()

	require.False(t, collection.IsWritable("https://anyone.test/@x"))
	require.True(t, collection.NotWritable("https://anyone.test/@x"))
	require.False(t, collection.IsWritable(""))
}

// Read and Write are independent: membership in one does not grant the other.
func TestCollection_ReadWrite_Independent(t *testing.T) {

	const reader = "https://reader.test/@r"
	const writer = "https://writer.test/@w"

	collection := NewCollection()
	collection.Read = sliceof.String{reader}
	collection.Write = sliceof.String{writer}

	require.True(t, collection.IsReadable(reader))
	require.False(t, collection.IsWritable(reader))

	require.True(t, collection.IsWritable(writer))
	require.False(t, collection.IsReadable(writer))
}

/******************************************
 * Getter / Setter Interfaces
 ******************************************/

// TestCollection_GetPointer confirms that every schema field resolves to a pointer, and unknown names do not
func TestCollection_GetPointer(t *testing.T) {

	collection := NewCollection()
	collection.ParentType = "Stream"
	collection.CollectionType = "Context"
	collection.Read = sliceof.String{"https://alice.test/@alice"}
	collection.Write = sliceof.String{"https://bob.test/@bob"}

	// "parentType" returns a pointer to the ParentType field
	{
		pointer, ok := collection.GetPointer("parentType")
		require.True(t, ok)
		parentTypePointer, ok := pointer.(*string)
		require.True(t, ok)
		require.Equal(t, "Stream", *parentTypePointer)
	}

	// "collectionType" returns a pointer to the CollectionType field
	{
		pointer, ok := collection.GetPointer("collectionType")
		require.True(t, ok)
		collectionTypePointer, ok := pointer.(*string)
		require.True(t, ok)
		require.Equal(t, "Context", *collectionTypePointer)
	}

	// "read" returns a pointer to the Read slice
	{
		pointer, ok := collection.GetPointer("read")
		require.True(t, ok)
		readPointer, ok := pointer.(*sliceof.String)
		require.True(t, ok)
		require.Equal(t, sliceof.String{"https://alice.test/@alice"}, *readPointer)
	}

	// "write" returns a pointer to the Write slice
	{
		pointer, ok := collection.GetPointer("write")
		require.True(t, ok)
		writePointer, ok := pointer.(*sliceof.String)
		require.True(t, ok)
		require.Equal(t, sliceof.String{"https://bob.test/@bob"}, *writePointer)
	}

	// An unknown property returns (nil, false)
	pointer, ok := collection.GetPointer("unknown-property")
	require.Nil(t, pointer)
	require.False(t, ok)
}

// TestCollection_GetStringOK confirms that the string fields read back, and unknown names report FALSE
func TestCollection_GetStringOK(t *testing.T) {

	collectionID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	parentID := primitive.NewObjectID()

	collection := NewCollection()
	collection.CollectionID = collectionID
	collection.UserID = userID
	collection.ParentID = parentID
	collection.ParentType = "Stream"
	collection.CollectionType = "Context"

	// Each of the three ObjectID fields returns its hex encoding with ok == true.
	{
		value, ok := collection.GetStringOK("collectionId")
		require.True(t, ok)
		require.Equal(t, collectionID.Hex(), value)
	}
	{
		value, ok := collection.GetStringOK("userId")
		require.True(t, ok)
		require.Equal(t, userID.Hex(), value)
	}
	{
		value, ok := collection.GetStringOK("parentId")
		require.True(t, ok)
		require.Equal(t, parentID.Hex(), value)
	}

	// The two string "type" fields return their raw values with ok == true.
	{
		value, ok := collection.GetStringOK("parentType")
		require.True(t, ok)
		require.Equal(t, "Stream", value)
	}
	{
		value, ok := collection.GetStringOK("collectionType")
		require.True(t, ok)
		require.Equal(t, "Context", value)
	}

	// An unknown property returns ("", false)
	value, ok := collection.GetStringOK("unknown-property")
	require.Equal(t, "", value)
	require.False(t, ok)
}

// TestCollection_SetString confirms that the string fields write through
func TestCollection_SetString(t *testing.T) {

	collectionID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	parentID := primitive.NewObjectID()

	collection := NewCollection()

	// A valid hex string sets each ObjectID field and returns true.
	require.True(t, collection.SetString("collectionId", collectionID.Hex()))
	require.Equal(t, collectionID, collection.CollectionID)

	require.True(t, collection.SetString("userId", userID.Hex()))
	require.Equal(t, userID, collection.UserID)

	require.True(t, collection.SetString("parentId", parentID.Hex()))
	require.Equal(t, parentID, collection.ParentID)
}

// TestCollection_SetString_Invalid confirms that an unparseable value is rejected rather than stored
func TestCollection_SetString_Invalid(t *testing.T) {

	collection := NewCollection()
	original := collection.CollectionID

	// A non-hex value leaves each field unchanged and returns false.
	require.False(t, collection.SetString("collectionId", "not-a-valid-object-id"))
	require.Equal(t, original, collection.CollectionID)

	require.False(t, collection.SetString("userId", "not-a-valid-object-id"))
	require.True(t, collection.UserID.IsZero())

	require.False(t, collection.SetString("parentId", "not-a-valid-object-id"))
	require.True(t, collection.ParentID.IsZero())

	// An unknown property returns false.
	require.False(t, collection.SetString("unknown-property", "value"))
}

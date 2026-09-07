package upgrades

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newFollowerAddressRecord builds one EMAIL Follower row as the planner reads it
func newFollowerAddressRecord(parentID primitive.ObjectID, emailAddress string, profileURL string) followerAddressRecord {

	result := followerAddressRecord{
		FollowerID: primitive.NewObjectID(),
		ParentID:   parentID,
	}

	result.Actor.EmailAddress = emailAddress
	result.Actor.ProfileURL = profileURL

	return result
}

/******************************************
 * planFollowerAddressNormalization
 ******************************************/

// A record already in normalized form is left alone, which is what makes a second run a no-op
func TestPlanFollowerAddressNormalization_AlreadyNormalizedIsUntouched(t *testing.T) {

	records := []followerAddressRecord{
		newFollowerAddressRecord(primitive.NewObjectID(), "sarah@connor.mil", "sarah@connor.mil"),
	}

	updates, collisions := planFollowerAddressNormalization(records)

	require.Empty(t, updates)
	require.Zero(t, collisions)
}

// Both address-bearing fields are rewritten together, because an EMAIL Follower carries the
// address in profileUrl too and LoadByActor matches on that one
func TestPlanFollowerAddressNormalization_RewritesBothFields(t *testing.T) {

	record := newFollowerAddressRecord(primitive.NewObjectID(), " Sarah@Connor.MIL ", "Sarah@Connor.MIL")

	updates, collisions := planFollowerAddressNormalization([]followerAddressRecord{record})

	require.Len(t, updates, 1)
	require.Equal(t, record.FollowerID, updates[0].FollowerID)
	require.Equal(t, "sarah@connor.mil", updates[0].EmailAddress)
	require.Equal(t, "sarah@connor.mil", updates[0].ProfileURL)
	require.Zero(t, collisions)
}

// A record whose profileUrl alone has drifted in case is still rewritten
func TestPlanFollowerAddressNormalization_ProfileURLAloneIsEnough(t *testing.T) {

	record := newFollowerAddressRecord(primitive.NewObjectID(), "sarah@connor.mil", "Sarah@connor.mil")

	updates, _ := planFollowerAddressNormalization([]followerAddressRecord{record})

	require.Len(t, updates, 1)
}

// Two records under one parent that differ only in case are both rewritten AND both counted,
// so the operator hears about the pair that this migration has just made identical
func TestPlanFollowerAddressNormalization_CountsCollisions(t *testing.T) {

	parentID := primitive.NewObjectID()

	records := []followerAddressRecord{
		newFollowerAddressRecord(parentID, "sarah@connor.mil", "sarah@connor.mil"),
		newFollowerAddressRecord(parentID, "SARAH@connor.mil", "SARAH@connor.mil"),
	}

	updates, collisions := planFollowerAddressNormalization(records)

	require.Len(t, updates, 1, "only the mixed-case record needs rewriting")
	require.Equal(t, 2, collisions, "but both records are part of the collision")
}

// The same address under two DIFFERENT parents is two different subscriptions, not a collision
func TestPlanFollowerAddressNormalization_DifferentParentsDoNotCollide(t *testing.T) {

	records := []followerAddressRecord{
		newFollowerAddressRecord(primitive.NewObjectID(), "sarah@connor.mil", "sarah@connor.mil"),
		newFollowerAddressRecord(primitive.NewObjectID(), "SARAH@connor.mil", "SARAH@connor.mil"),
	}

	_, collisions := planFollowerAddressNormalization(records)

	require.Zero(t, collisions)
}

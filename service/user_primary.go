package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
)

// applyPrimaryUser keeps the primary-user rules before a User is saved: only an owner can be
// the primary user, and a User who becomes primary replaces the one before
func (service *User) applyPrimaryUser(session data.Session, user *model.User) error {

	const location = "service.User.applyPrimaryUser"

	// RULE: Only an owner can be the primary user, so a demoted owner loses the flag on the same save
	if !user.IsOwner {
		user.IsPrimary = false
	}

	// Nothing more to do for anyone else
	if !user.IsPrimary {
		return nil
	}

	// Find any other primary user.  idx_User_Primary refuses a second one if two servers race past this read
	others, err := service.Query(session, exp.Equal("isPrimary", true).AndNotEqual("_id", user.UserID))

	if err != nil {
		return derp.Wrap(err, location, "Loading the current primary user", user.UserID)
	}

	// RULE: A new primary user replaces the old one
	for index := range others {

		others[index].IsPrimary = false

		if err := service.Save(session, &others[index], "Primary user moved to "+user.UserID.Hex()); err != nil {
			return derp.Wrap(err, location, "Clearing the previous primary user", others[index].UserID)
		}
	}

	// There can be only one
	return nil
}

// recordPrimaryUser keeps Domain.PrimaryUserID in step with a User that was just saved or deleted
func (service *User) recordPrimaryUser(session data.Session, user *model.User) error {

	const location = "service.User.recordPrimaryUser"

	// A User who represents the domain is its primary user
	if user.RepresentsDomain() {

		if err := service.domainService.SetPrimaryUser(session, user.UserID); err != nil {
			return derp.Wrap(err, location, "Recording the primary user", user.UserID)
		}

		return nil
	}

	// Anyone else is not, so the domain forgets this User if it named them
	if err := service.domainService.ClearPrimaryUser(session, user.UserID); err != nil {
		return derp.Wrap(err, location, "Clearing the primary user", user.UserID)
	}

	// The king is dead, long live the king
	return nil
}

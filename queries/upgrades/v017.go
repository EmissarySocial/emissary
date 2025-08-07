package upgrades

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// Version17...
func Version17(ctx context.Context, session *mongo.Database) error {

	fmt.Println("... Version 17")

	/*
		if err := MakeIndex(session, "Attachment", "idx_Attachment_Object", "objectType", "objectId", "category"); err != nil {
			return err
		}

		if err := MakeIndex(session, "EncryptionKey", "idx_EncryptionKey_Parent", "parentType", "parentId"); err != nil {
			return err
		}

		if err := MakeIndex(session, "Folder", "idx_Foler_User", "userId", "rank"); err != nil {
			return err
		}

		if err := MakeIndex(session, "Follower", "idx_Follower_Parent", "parentType", "parentId"); err != nil {
			return err
		}

		if err := MakeIndex(session, "Following", "idx_Following_User", "userId", "folderId"); err != nil {
			return err
		}

		if err := MakeIndex(session, "Inbox", "idx_Inbox_User", "userId", "folderId", "readDate"); err != nil {
			return err
		}

		if err := MakeIndex(session, "JWT", "idx_JWT_Key", "keyName"); err != nil {
			return err
		}

		if err := MakeIndex(session, "Mention", "idx_Mention_Object", "objectId", "type"); err != nil {
			return err
		}

		if err := MakeIndex(session, "Outbox", "idx_Outbox_Parent", "parentType", "parentId"); err != nil {
			return err
		}

		if err := MakeIndex(session, "Response", "idx_Response_Object", "object", "userId"); err != nil {
			return err
		}

		if err := MakeIndex(session, "Rule", "idx_Rule_User", "userId", "followingId"); err != nil {
			return err
		}

		if err := MakeIndex(session, "Stream", "idx_Stream_ParentID", "parentId", "rank"); err != nil {
			return err
		}

		if err := MakeIndex(session, "Stream", "idx_Stream_Token", "token"); err != nil {
			return err
		}

		if err := MakeIndex(session, "User", "idx_User_Username", "username"); err != nil {
			return err
		}
	*/
	return nil
}

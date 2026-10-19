package sync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// insertTestAnnotation writes one raw Annotation document, bypassing the service layer.
func insertTestAnnotation(t *testing.T, database *mongo.Database, userID primitive.ObjectID, url string, content string, createDate int64, deleteDate int64) primitive.ObjectID {

	t.Helper()

	annotationID := primitive.NewObjectID()

	_, err := database.Collection("Annotation").InsertOne(context.Background(), bson.M{
		"_id":        annotationID,
		"userId":     userID,
		"url":        url,
		"content":    content,
		"createDate": createDate,
		"deleteDate": deleteDate,
	})

	require.Nil(t, err)
	return annotationID
}

// liveAnnotationIDs returns the IDs of every live Annotation in the test database.
func liveAnnotationIDs(t *testing.T, database *mongo.Database) []primitive.ObjectID {

	t.Helper()

	cursor, err := database.Collection("Annotation").Find(context.Background(), bson.M{"deleteDate": 0})
	require.Nil(t, err)

	records := make([]struct {
		AnnotationID primitive.ObjectID `bson:"_id"`
	}, 0)

	require.Nil(t, cursor.All(context.Background(), &records))

	result := make([]primitive.ObjectID, 0, len(records))
	for _, record := range records {
		result = append(result, record.AnnotationID)
	}

	return result
}

func TestAnnotation_KeepsTheLongestNoteAndBuildsTheIndex(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	userID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()

	// Three annotations of one URL by one User, and unrelated records that must survive
	short := insertTestAnnotation(t, database, userID, "https://example.test/a", "short", 1, 0)
	long := insertTestAnnotation(t, database, userID, "https://example.test/a", "the longest note", 2, 0)
	empty := insertTestAnnotation(t, database, userID, "https://example.test/a", "", 3, 0)
	other := insertTestAnnotation(t, database, otherUserID, "https://example.test/a", "", 4, 0)
	deleted := insertTestAnnotation(t, database, userID, "https://example.test/a", "an even longer deleted note", 5, 99)

	require.Nil(t, Annotation(ctx, database))

	live := liveAnnotationIDs(t, database)
	require.ElementsMatch(t, []primitive.ObjectID{long, other}, live)
	require.NotContains(t, live, short)
	require.NotContains(t, live, empty)
	require.NotContains(t, live, deleted)

	exists, err := indexExists(ctx, database.Collection("Annotation"), annotationUniqueIndex)
	require.Nil(t, err)
	require.True(t, exists, "the unique index must build once duplicates are gone")
}

func TestAnnotation_IndexRefusesASecondLiveAnnotation(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()
	userID := primitive.NewObjectID()

	require.Nil(t, Annotation(ctx, database))

	insertTestAnnotation(t, database, userID, "https://example.test/a", "", 1, 0)

	// A soft-deleted row never collides, so a removed annotation can be made again
	insertTestAnnotation(t, database, userID, "https://example.test/a", "", 2, 99)

	_, err := database.Collection("Annotation").InsertOne(ctx, bson.M{
		"_id":        primitive.NewObjectID(),
		"userId":     userID,
		"url":        "https://example.test/a",
		"deleteDate": 0,
	})

	require.True(t, mongo.IsDuplicateKeyError(err), "a second live annotation of one URL must be refused")
}

func TestAnnotation_SyncIsStable(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.Nil(t, Annotation(ctx, database))
	require.Nil(t, Annotation(ctx, database))

	cursor, err := database.Collection("Annotation").Indexes().List(ctx)
	require.Nil(t, err)

	indexes := make([]bson.M, 0)
	require.Nil(t, cursor.All(ctx, &indexes))

	names := make([]string, 0, len(indexes))
	for _, index := range indexes {
		names = append(names, index["name"].(string))
	}

	require.ElementsMatch(t, []string{"_id_", "idx_Annotation_Recycle", annotationUniqueIndex, "idx_Annotation_User_Created"}, names)
}

func TestSurvivingAnnotationID(t *testing.T) {

	first := primitive.NewObjectID()
	second := primitive.NewObjectID()

	// Longest note wins
	require.Equal(t, second, survivingAnnotationID([]annotationRecord{
		{AnnotationID: first, ContentLength: 1, CreateDate: 1},
		{AnnotationID: second, ContentLength: 5, CreateDate: 2},
	}))

	// A tie keeps the earliest
	require.Equal(t, first, survivingAnnotationID([]annotationRecord{
		{AnnotationID: second, ContentLength: 3, CreateDate: 2},
		{AnnotationID: first, ContentLength: 3, CreateDate: 1},
	}))
}

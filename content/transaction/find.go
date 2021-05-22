package transaction

import "github.com/benpate/ghost/content"

// findParent returns the item that contains the provided itemID.  If no
// container is found, then (-1, nil) is returned
func findParent(c *content.Content, itemID int) (int, *content.Item) {

	for itemIndex := range *c {
		if refIndex := findChildPosition(c, itemIndex, itemID); refIndex != -1 {
			return itemIndex, &(*c)[itemIndex]
		}
	}

	return -1, nil
}

// findChildPosition returns the position of a childID in the ref array.
func findChildPosition(c *content.Content, itemID int, childID int) int {
	for refIndex := range (*c)[itemID].Refs {
		if (*c)[itemID].Refs[refIndex] == childID {
			return refIndex
		}
	}

	return -1
}

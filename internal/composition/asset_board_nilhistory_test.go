package composition

import (
	"context"
	"testing"
)

// A fresh profile has no history store yet; publishing the asset model must wait, not crash.
func TestPublishAssetModelWithoutHistory(t *testing.T) {
	t.Parallel()
	a, _ := byeFixtureApp(t)
	a.core.History = nil
	a.publishAssetModel(context.Background())
}

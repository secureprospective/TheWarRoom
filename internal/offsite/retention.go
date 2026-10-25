package offsite

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

const objectFormat = "20060102T150405.000000000Z"

func objectTime(name string) (time.Time, error) {
	if !strings.HasPrefix(name, "sets/") || !strings.HasSuffix(name, ".tar.gz.age") {
		return time.Time{}, fmt.Errorf("not a recovery set")
	}
	at, err := time.Parse(objectFormat, strings.TrimSuffix(strings.TrimPrefix(name, "sets/"), ".tar.gz.age"))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid recovery set name: %w", err)
	}
	return at, nil
}
func (r *R2) pruneObjects(ctx context.Context, objects []minio.ObjectInfo, now time.Time) error {
	slices.SortFunc(objects, func(a, b minio.ObjectInfo) int { return strings.Compare(b.Key, a.Key) })
	days := map[string]bool{}
	for i, object := range objects {
		at, err := objectTime(object.Key)
		if err != nil {
			return err
		}
		day := at.UTC().Format(time.DateOnly)
		keep := i < 12 || (!days[day] && !at.Before(now.Add(-30*24*time.Hour))) || at.After(now)
		days[day] = true
		if keep {
			continue
		}
		if err := r.client.RemoveObject(ctx, r.bucket, object.Key, minio.RemoveObjectOptions{}); err != nil {
			return fmt.Errorf("R2 retention could not remove an expired backup")
		}
	}
	return nil
}

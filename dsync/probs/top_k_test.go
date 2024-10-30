package probs_test

import (
	"testing"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/assert"
)

func TestTopK(t *testing.T) {
	client := redistest.Client(t)
	// Find top k hashtag
	topK := probs.NewTopK(client)
	key := t.Name() + ":top_k:hashtag"

	is := assert.New(t)
	is.Nil(topK.Create(ctx, key, 5))
	topK.Add(ctx, key, "#ai", "#ml", "#js")
}

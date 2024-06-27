package memory

import (
	"github.com/viciious/mika/store"
	"testing"
)

func TestMemoryTorrentStore(t *testing.T) {
	store.TestStore(t, NewDriver())
}

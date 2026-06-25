package kernel

import (
	"testing"

	"github.com/cedar2025/xboard-node/internal/model"
)

func TestUserDiffTreatsUUIDChangeAsRemoveAndAdd(t *testing.T) {
	oldUsers := []model.UserSpec{{ID: 1, UUID: "old-uuid"}}
	newUsers := []model.UserSpec{{ID: 1, UUID: "new-uuid"}}

	toAdd, toRemove := UserDiff(oldUsers, newUsers)

	if len(toAdd) != 1 || toAdd[0].UUID != "new-uuid" {
		t.Fatalf("toAdd = %#v, want new uuid", toAdd)
	}
	if len(toRemove) != 1 || toRemove[0].UUID != "old-uuid" {
		t.Fatalf("toRemove = %#v, want old uuid", toRemove)
	}
}

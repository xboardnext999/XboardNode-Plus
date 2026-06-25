package limiter

import (
	"sort"
	"testing"

	"github.com/cedar2025/xboard-node/internal/model"
)

func TestNew(t *testing.T) {
	l := New()
	if l == nil {
		t.Fatal("New returned nil")
	}
}

func TestUpdateUsers(t *testing.T) {
	l := New()
	users := []model.UserSpec{
		{ID: 1, UUID: "u1", SpeedLimit: 3, DeviceLimit: 2},
		{ID: 2, UUID: "u2", SpeedLimit: 0, DeviceLimit: 0},
	}
	removed := l.UpdateUsers(users)

	if len(removed) != 0 {
		t.Errorf("first update should have no removed users, got %v", removed)
	}
}

func TestUpdateUsers_DetectsRemoved(t *testing.T) {
	l := New()
	l.UpdateUsers([]model.UserSpec{
		{ID: 1, UUID: "u1"},
		{ID: 2, UUID: "u2"},
		{ID: 3, UUID: "u3"},
	})

	// Remove user 2
	removed := l.UpdateUsers([]model.UserSpec{
		{ID: 1, UUID: "u1"},
		{ID: 3, UUID: "u3"},
	})

	if len(removed) != 1 || removed[0] != 2 {
		t.Errorf("expected removed=[2], got %v", removed)
	}
}

func TestUpdateUsers_DetectsMultipleRemoved(t *testing.T) {
	l := New()
	l.UpdateUsers([]model.UserSpec{
		{ID: 1, UUID: "u1"},
		{ID: 2, UUID: "u2"},
		{ID: 3, UUID: "u3"},
	})

	// Remove users 1 and 3
	removed := l.UpdateUsers([]model.UserSpec{
		{ID: 2, UUID: "u2"},
	})

	sort.Ints(removed)
	if len(removed) != 2 || removed[0] != 1 || removed[1] != 3 {
		t.Errorf("expected removed=[1,3], got %v", removed)
	}
}

func TestUpdateUsers_NoRemovals(t *testing.T) {
	l := New()
	l.UpdateUsers([]model.UserSpec{
		{ID: 1, UUID: "u1"},
	})

	// Add user, keep existing
	removed := l.UpdateUsers([]model.UserSpec{
		{ID: 1, UUID: "u1"},
		{ID: 2, UUID: "u2"},
	})

	if len(removed) != 0 {
		t.Errorf("expected no removals, got %v", removed)
	}
}

func TestUpdateUsers_ReplacesAll(t *testing.T) {
	l := New()
	l.UpdateUsers([]model.UserSpec{{ID: 1, DeviceLimit: 1}})

	// Replace with different users
	removed := l.UpdateUsers([]model.UserSpec{{ID: 2, DeviceLimit: 1}})

	if len(removed) != 1 || removed[0] != 1 {
		t.Errorf("expected removed=[1], got %v", removed)
	}
}

func TestGetDeviceLimitByUUID(t *testing.T) {
	l := New()
	l.UpdateUsers([]model.UserSpec{
		{ID: 1, UUID: "u1", DeviceLimit: 3},
		{ID: 2, UUID: "u2", DeviceLimit: 0},
	})

	// User with device limit
	limit, ok := l.GetDeviceLimitByUUID("u1")
	if !ok || limit != 3 {
		t.Errorf("expected (3, true), got (%d, %v)", limit, ok)
	}

	// User without device limit
	_, ok = l.GetDeviceLimitByUUID("u2")
	if ok {
		t.Error("expected (_, false) for user with no device limit")
	}

	// Unknown UUID
	_, ok = l.GetDeviceLimitByUUID("unknown")
	if ok {
		t.Error("expected (_, false) for unknown UUID")
	}
}

func TestSnapshotMetrics(t *testing.T) {
	l := New()
	m := l.SnapshotMetrics()
	if m.DeviceLimitEvents != 0 {
		t.Errorf("expected 0 events initially, got %d", m.DeviceLimitEvents)
	}
}

package ui

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/helgesverre/dbdump/internal/database"
)

func TestTableSelectionModelCancelMarksSelectionCancelled(t *testing.T) {
	t.Parallel()

	model := NewTableSelectionModel([]database.TableInfo{{Name: "users"}}, nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	got := updated.(TableSelectionModel)

	if !got.cancelled {
		t.Fatal("expected model to be marked cancelled")
	}

	if !got.done {
		t.Fatal("expected model to be marked done")
	}
}

func TestTableSelectionModelSpaceOnEmptyTablesIsSafe(t *testing.T) {
	t.Parallel()

	model := NewTableSelectionModel(nil, nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	got := updated.(TableSelectionModel)

	if len(got.selected) != 0 {
		t.Fatalf("expected no selections, got %#v", got.selected)
	}
}

func TestGetSelectedReturnsSortedTableNames(t *testing.T) {
	t.Parallel()

	model := NewTableSelectionModel(nil, nil)
	model.selected["zeta"] = true
	model.selected["alpha"] = true

	got := model.GetSelected()
	want := []string{"alpha", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected selected tables: got %v want %v", got, want)
	}
}

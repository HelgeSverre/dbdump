package ui

import (
	"errors"
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

func TestTableSelectionModelEnterMarksConfirmed(t *testing.T) {
	t.Parallel()

	model := NewTableSelectionModel([]database.TableInfo{{Name: "users"}}, nil)
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(TableSelectionModel)

	if !got.confirmed {
		t.Fatal("expected enter to mark the selection confirmed")
	}
	if got.cancelled {
		t.Fatal("did not expect enter to mark the selection cancelled")
	}
}

func TestSelectionResultTreatsUnconfirmedExitAsCancelled(t *testing.T) {
	t.Parallel()

	// A SIGTERM makes bubbletea return (model, nil) without running our key
	// handler, so the model is neither confirmed nor cancelled.
	model := NewTableSelectionModel([]database.TableInfo{{Name: "users"}}, []string{"users"})

	if _, err := selectionResult(model, nil); !errors.Is(err, ErrSelectionCancelled) {
		t.Fatalf("expected unconfirmed exit to be treated as cancelled, got %v", err)
	}
}

func TestSelectionResultTreatsInterruptAsCancelled(t *testing.T) {
	t.Parallel()

	model := NewTableSelectionModel(nil, nil)
	for _, runErr := range []error{tea.ErrInterrupted, tea.ErrProgramKilled} {
		if _, err := selectionResult(model, runErr); !errors.Is(err, ErrSelectionCancelled) {
			t.Fatalf("expected %v to be treated as cancelled, got %v", runErr, err)
		}
	}
}

func TestSelectionResultReturnsSelectionWhenConfirmed(t *testing.T) {
	t.Parallel()

	model := NewTableSelectionModel(nil, nil)
	model.confirmed = true
	model.selected["audits"] = true

	got, err := selectionResult(model, nil)
	if err != nil {
		t.Fatalf("selectionResult returned error: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"audits"}) {
		t.Fatalf("unexpected selection: %v", got)
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

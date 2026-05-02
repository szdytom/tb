package tui

import (
	"fmt"
	"strings"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"github.com/szdytom/tb/internal/buffer"
	"github.com/szdytom/tb/internal/store"
)

// searchResult is posted asynchronously after a debounced search completes.
type searchResult struct {
	summaries []buffer.BufferSummary
	gen       int
	err       error
}

func (a *App) enterSearch() {
	a.searchGen = 0
	a.searchInput = newStatusInput("/")
	// Save pre-search state so Escape can restore it.
	a.savedSearchQuery = a.searchQuery
	a.savedSearchFilter = append([]buffer.BufferSummary{}, a.summaries...)
	// allSummaries always tracks the full unfiltered list.
	if a.searchQuery == "" {
		a.allSummaries = a.summaries
	}

	a.curState = stateSearch
}

func (a *App) exitSearch() {
	if a.searchTimer != nil {
		a.searchTimer.Stop()
		a.searchTimer = nil
	}

	a.searchInput = nil
	a.searchGen++

	// Restore the state that was active when search was entered.
	if a.savedSearchQuery != "" {
		a.searchQuery = a.savedSearchQuery
		a.summaries = a.savedSearchFilter
	} else {
		a.summaries = a.allSummaries
		a.searchQuery = ""
	}

	if a.cursor >= len(a.summaries) {
		a.cursor = 0
	}

	if a.listOff > a.cursor {
		a.listOff = a.cursor
	}

	a.curState = stateBrowsing
	a.vx.HideCursor()

	if len(a.summaries) > 0 {
		a.loadPreviewAsync()
	}
}

func (a *App) commitSearch() {
	if a.searchTimer != nil {
		a.searchTimer.Stop()
		a.searchTimer = nil
	}

	var query string
	if a.searchInput != nil {
		query = a.searchInput.String()
	}

	a.searchInput = nil

	if query == "" {
		// Empty query → dismiss search prompt, keep current filter intact
		a.curState = stateBrowsing
		a.vx.HideCursor()

		return
	}

	a.searchQuery = query
	a.curState = stateBrowsing
	a.vx.HideCursor()

	// Launch the search immediately — the debounce timer may not have fired yet.
	a.searchGen++
	a.doSearch(query, a.searchGen)
}

// clearFilter restores the full buffer list when a search filter is active.
func (a *App) clearFilter() {
	a.summaries = a.allSummaries

	a.searchQuery = ""
	if a.cursor >= len(a.summaries) {
		a.cursor = 0
	}

	if a.listOff > a.cursor {
		a.listOff = a.cursor
	}

	if len(a.summaries) > 0 {
		a.loadPreviewAsync()
	}
}

func (a *App) searchDebounced() {
	if a.searchTimer != nil {
		a.searchTimer.Stop()
	}

	a.searchGen++
	gen := a.searchGen

	var query string
	if a.searchInput != nil {
		query = a.searchInput.String()
	}

	a.searchTimer = time.AfterFunc(150*time.Millisecond, func() {
		a.doSearch(query, gen)
	})
}

// triggerSearch runs a search immediately (no debounce). Used when buffers are
// created or deleted while in search mode.
func (a *App) triggerSearch() {
	a.searchGen++
	gen := a.searchGen

	var query string
	if a.searchInput != nil {
		query = a.searchInput.String()
	}

	a.doSearch(query, gen)
}

func (a *App) doSearch(query string, gen int) {
	if query == "" {
		return
	}

	go func() {
		var (
			summaries []buffer.BufferSummary
			err       error
		)

		var results []store.SearchResult
		if strings.HasPrefix(query, "~") {
			results, err = a.client.Search(query[1:], "regex")
		} else {
			results, err = a.client.Search(query, "fuzzy")
		}

		if err == nil {
			for _, r := range results {
				summaries = append(summaries, buffer.NewBufferSummary(r.Buffer))
			}
		}

		if err != nil {
			a.vx.PostEvent(searchResult{err: err, gen: gen})

			return
		}

		if summaries == nil {
			summaries = []buffer.BufferSummary{}
		}

		a.vx.PostEvent(searchResult{summaries: summaries, gen: gen})
	}()
}

func (a *App) handleKeySearch(ev vaxis.Key) {
	action, val := a.searchInput.HandleKey(ev)

	switch action {
	case inputCancel:
		a.exitSearch()

		return
	case inputCommit:
		a.commitSearch()

		return
	case inputChanged:
		if val == "" {
			a.summaries = a.allSummaries
			if a.searchTimer != nil {
				a.searchTimer.Stop()
				a.searchTimer = nil
			}

			a.searchGen++
		} else {
			a.searchDebounced()
		}
	}
}

func (a *App) handleSearchResult(msg searchResult) {
	if msg.err != nil {
		a.setError("Search failed: " + msg.err.Error())

		return
	}

	if msg.gen != a.searchGen {
		return
	}

	a.summaries = msg.summaries
	if a.cursor >= len(a.summaries) {
		a.cursor = 0
	}

	if a.listOff > a.cursor {
		a.listOff = a.cursor
	}

	if len(a.summaries) > 0 {
		a.loadPreviewAsync()
	}
}

func (a *App) drawSearchBar(win vaxis.Window) {
	if a.searchInput == nil {
		return
	}

	a.searchInput.Draw(win)
}

// searchStatusText returns a status bar message when a search filter is active.
func (a *App) searchStatusText() string {
	if a.searchQuery != "" {
		return fmt.Sprintf(" Searching: %s  (Esc/Ctrl-C to clear) ", a.searchQuery)
	}

	return ""
}

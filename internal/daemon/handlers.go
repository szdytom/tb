package daemon

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/szdytom/tb/internal/buffer"
	"github.com/szdytom/tb/internal/ipc"
	"github.com/szdytom/tb/internal/store"
)

// dispatch routes a request to the appropriate handler based on Op.
func (d *Daemon) dispatch(req *ipc.Request) *ipc.Response {
	switch req.Op {
	case ipc.OpPing:
		return d.handlePing(req)
	case ipc.OpCreateBuffer:
		return d.handleCreateBuffer(req)
	case ipc.OpGetBuffer:
		return d.handleGetBuffer(req)
	case ipc.OpListBuffers:
		return d.handleListBuffers(req)
	case ipc.OpUpdateContent:
		return d.handleUpdateContent(req)
	case ipc.OpUpdateLabel:
		return d.handleUpdateLabel(req)
	case ipc.OpUpdateTags:
		return d.handleUpdateTags(req)
	case ipc.OpSoftDelete:
		return d.handleSoftDelete(req)
	case ipc.OpPermanentlyDelete:
		return d.handlePermanentlyDelete(req)
	case ipc.OpListTrash:
		return d.handleListTrash(req)
	case ipc.OpRestoreFromTrash:
		return d.handleRestoreFromTrash(req)
	case ipc.OpSearch:
		return d.handleSearch(req)
	case ipc.OpCount:
		return d.handleCount(req)
	default:
		return ptr(ipc.ErrorResponse(req.ID, fmt.Sprintf("unknown operation: %s", req.Op)))
	}
}

func (d *Daemon) handlePing(req *ipc.Request) *ipc.Response {
	resp := ipc.OKResponse(req.ID, ipc.PingResponse{Message: "pong"})
	return &resp
}

func (d *Daemon) handleCreateBuffer(req *ipc.Request) *ipc.Response {
	var p ipc.CreateBufferPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		return ptr(ipc.ErrorResponse(req.ID, fmt.Sprintf("invalid payload: %v", err)))
	}
	buf := buffer.NewBuffer(p.Content, p.Label, p.Tags)
	id, err := d.repo.Insert(buf)
	if err != nil {
		log.Printf("create buffer: %v", err)
		return ptr(ipc.ErrorResponse(req.ID, err.Error()))
	}
	return ptr(ipc.OKResponse(req.ID, ipc.IDResponse{ID: id}))
}

func (d *Daemon) handleGetBuffer(req *ipc.Request) *ipc.Response {
	var p ipc.IDPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		return ptr(ipc.ErrorResponse(req.ID, fmt.Sprintf("invalid payload: %v", err)))
	}
	buf, err := d.repo.Get(p.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ptr(ipc.ErrorResponse(req.ID, "buffer not found"))
		}
		log.Printf("get buffer %d: %v", p.ID, err)
		return ptr(ipc.ErrorResponse(req.ID, err.Error()))
	}
	return ptr(ipc.OKResponse(req.ID, buf))
}

func (d *Daemon) handleListBuffers(req *ipc.Request) *ipc.Response {
	var p ipc.ListBuffersPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		return ptr(ipc.ErrorResponse(req.ID, fmt.Sprintf("invalid payload: %v", err)))
	}

	filter := store.ListFilter{
		Keyword: p.Keyword,
		Limit:   p.Limit,
		Offset:  p.Offset,
		SortAsc: p.SortAsc,
	}

	if p.Since != "" {
		t, err := time.Parse(time.RFC3339, p.Since)
		if err != nil {
			return ptr(ipc.ErrorResponse(req.ID, fmt.Sprintf("invalid since: %v", err)))
		}
		filter.Since = &t
	}
	if p.Until != "" {
		t, err := time.Parse(time.RFC3339, p.Until)
		if err != nil {
			return ptr(ipc.ErrorResponse(req.ID, fmt.Sprintf("invalid until: %v", err)))
		}
		filter.Until = &t
	}
	switch store.SortField(p.SortBy) {
	case store.SortByCreatedAt, store.SortByLabel, store.SortByID:
		filter.SortBy = store.SortField(p.SortBy)
	default:
		filter.SortBy = store.SortByUpdatedAt
	}

	bufs, err := d.repo.List(filter)
	if err != nil {
		log.Printf("list buffers: %v", err)
		return ptr(ipc.ErrorResponse(req.ID, err.Error()))
	}
	if bufs == nil {
		bufs = []*buffer.Buffer{}
	}
	return ptr(ipc.OKResponse(req.ID, bufs))
}

func (d *Daemon) handleUpdateContent(req *ipc.Request) *ipc.Response {
	var p ipc.UpdateContentPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		return ptr(ipc.ErrorResponse(req.ID, fmt.Sprintf("invalid payload: %v", err)))
	}
	if err := d.repo.UpdateContent(p.ID, p.Content); err != nil {
		log.Printf("update content %d: %v", p.ID, err)
		return ptr(ipc.ErrorResponse(req.ID, err.Error()))
	}
	return ptr(ipc.OKResponse(req.ID, nil))
}

func (d *Daemon) handleUpdateLabel(req *ipc.Request) *ipc.Response {
	var p ipc.UpdateLabelPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		return ptr(ipc.ErrorResponse(req.ID, fmt.Sprintf("invalid payload: %v", err)))
	}
	if err := d.repo.UpdateLabel(p.ID, p.Label); err != nil {
		log.Printf("update label %d: %v", p.ID, err)
		return ptr(ipc.ErrorResponse(req.ID, err.Error()))
	}
	return ptr(ipc.OKResponse(req.ID, nil))
}

func (d *Daemon) handleUpdateTags(req *ipc.Request) *ipc.Response {
	var p ipc.UpdateTagsPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		return ptr(ipc.ErrorResponse(req.ID, fmt.Sprintf("invalid payload: %v", err)))
	}
	if err := d.repo.UpdateTags(p.ID, p.Tags); err != nil {
		log.Printf("update tags %d: %v", p.ID, err)
		return ptr(ipc.ErrorResponse(req.ID, err.Error()))
	}
	return ptr(ipc.OKResponse(req.ID, nil))
}

func (d *Daemon) handleSoftDelete(req *ipc.Request) *ipc.Response {
	var p ipc.SoftDeletePayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		return ptr(ipc.ErrorResponse(req.ID, fmt.Sprintf("invalid payload: %v", err)))
	}
	ttl := time.Duration(p.TTLSeconds) * time.Second
	if err := d.repo.SoftDelete(p.ID, ttl); err != nil {
		log.Printf("soft delete %d: %v", p.ID, err)
		return ptr(ipc.ErrorResponse(req.ID, err.Error()))
	}
	return ptr(ipc.OKResponse(req.ID, nil))
}

func (d *Daemon) handlePermanentlyDelete(req *ipc.Request) *ipc.Response {
	var p ipc.IDPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		return ptr(ipc.ErrorResponse(req.ID, fmt.Sprintf("invalid payload: %v", err)))
	}
	if err := d.repo.PermanentlyDelete(p.ID); err != nil {
		log.Printf("permanently delete %d: %v", p.ID, err)
		return ptr(ipc.ErrorResponse(req.ID, err.Error()))
	}
	return ptr(ipc.OKResponse(req.ID, nil))
}

func (d *Daemon) handleListTrash(req *ipc.Request) *ipc.Response {
	bufs, err := d.repo.ListTrash()
	if err != nil {
		log.Printf("list trash: %v", err)
		return ptr(ipc.ErrorResponse(req.ID, err.Error()))
	}
	if bufs == nil {
		bufs = []*buffer.Buffer{}
	}
	return ptr(ipc.OKResponse(req.ID, bufs))
}

func (d *Daemon) handleRestoreFromTrash(req *ipc.Request) *ipc.Response {
	var p ipc.IDPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		return ptr(ipc.ErrorResponse(req.ID, fmt.Sprintf("invalid payload: %v", err)))
	}
	if err := d.repo.RestoreFromTrash(p.ID); err != nil {
		log.Printf("restore from trash %d: %v", p.ID, err)
		return ptr(ipc.ErrorResponse(req.ID, err.Error()))
	}
	return ptr(ipc.OKResponse(req.ID, nil))
}

func (d *Daemon) handleSearch(req *ipc.Request) *ipc.Response {
	var p ipc.SearchPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		return ptr(ipc.ErrorResponse(req.ID, fmt.Sprintf("invalid payload: %v", err)))
	}
	results, err := d.repo.Search(p.Query, p.IsRegex)
	if err != nil {
		log.Printf("search: %v", err)
		return ptr(ipc.ErrorResponse(req.ID, err.Error()))
	}
	if results == nil {
		results = []store.SearchResult{}
	}
	return ptr(ipc.OKResponse(req.ID, results))
}

func (d *Daemon) handleCount(req *ipc.Request) *ipc.Response {
	n, err := d.repo.Count()
	if err != nil {
		log.Printf("count: %v", err)
		return ptr(ipc.ErrorResponse(req.ID, err.Error()))
	}
	return ptr(ipc.OKResponse(req.ID, ipc.CountResponse{Count: n}))
}

// ptr returns a pointer to the given value. Needed because dispatch
// returns *ipc.Response but the handlers return ipc.Response values.
func ptr[T any](v T) *T {
	return &v
}

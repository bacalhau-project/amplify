package db

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/bacalhau-project/amplify/pkg/util"
	"github.com/google/uuid"
	"golang.org/x/exp/constraints"
)

func NewInMemDB() Persistence {
	return &inMemDB{
		queueItems: make(map[uuid.UUID]QueueItem),
		nodes:      make(map[int32]Node),
		edges:      make(map[int32]Edge),
		ioSpecs:    make(map[int32]IoSpec),
		results:    make(map[int32]Result),
		statuses:   make(map[int32]Status),
	}
}

type inMemDB struct {
	mu            sync.RWMutex
	queueItems    map[uuid.UUID]QueueItem
	nodes         map[int32]Node
	nodeCounter   int32
	edges         map[int32]Edge
	edgeCounter   int32
	ioSpecs       map[int32]IoSpec
	ioSpecCounter int32
	results       map[int32]Result
	resultCounter int32
	statuses      map[int32]Status
	statusCounter int32
}

func (r *inMemDB) CreateQueueItem(ctx context.Context, arg CreateQueueItemParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.queueItems[arg.ID]
	if ok {
		return fmt.Errorf("queue item with id %s already exists", arg.ID)
	}
	r.queueItems[arg.ID] = QueueItem(arg)
	return nil
}

func (r *inMemDB) GetNodesByQueueItemID(ctx context.Context, queueItemID uuid.UUID) ([]Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	nodes := make([]Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		if n.QueueItemID == queueItemID {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

func (r *inMemDB) GetQueueItemDetail(ctx context.Context, id uuid.UUID) (QueueItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	i, ok := r.queueItems[id]
	if !ok {
		return QueueItem{}, ErrNotFound
	}
	return i, nil
}

func (r *inMemDB) ListQueueItems(ctx context.Context, arg ListQueueItemsParams) ([]QueueItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	queueItems := make([]QueueItem, 0, len(r.queueItems))
	for _, i := range r.queueItems {
		queueItems = append(queueItems, i)
	}
	sort.Slice(queueItems, func(i, j int) bool {
		return queueItems[i].CreatedAt.After(queueItems[j].CreatedAt)
	})
	return queueItems[0:min(len(queueItems), int(arg.Limit))], nil
}

func (r *inMemDB) CreateEdge(ctx context.Context, arg CreateEdgeParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.edgeCounter++
	r.edges[r.edgeCounter] = Edge{
		ID:       r.edgeCounter,
		ParentID: arg.ParentID,
		ChildID:  arg.ChildID,
	}
	return nil
}

func (r *inMemDB) CreateIOSpec(ctx context.Context, arg CreateIOSpecParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ioSpecCounter++
	r.ioSpecs[r.ioSpecCounter] = IoSpec{
		ID:       r.ioSpecCounter,
		NodeID:   arg.NodeID,
		Type:     arg.Type,
		NodeName: arg.NodeName,
		InputID:  arg.InputID,
		Root:     arg.Root,
		Value:    arg.Value,
		Path:     arg.Path,
		Context:  arg.Context,
	}
	return nil
}

func (r *inMemDB) CreateAndReturnNode(ctx context.Context, arg CreateAndReturnNodeParams) (Node, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodeCounter++
	r.nodes[r.nodeCounter] = Node{
		ID:          r.nodeCounter,
		QueueItemID: arg.QueueItemID,
		Name:        arg.Name,
	}
	return r.nodes[r.nodeCounter], nil
}

func (r *inMemDB) CreateResult(ctx context.Context, arg CreateResultParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resultCounter++
	r.results[r.resultCounter] = Result{
		ID:          r.resultCounter,
		NodeID:      arg.NodeID,
		ExecutionID: arg.ExecutionID,
		Stdout:      arg.Stdout,
		Stderr:      arg.Stderr,
		Skipped:     arg.Skipped,
	}
	return nil
}

func (r *inMemDB) CreateStatus(ctx context.Context, arg CreateStatusParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statusCounter++
	r.statuses[r.statusCounter] = Status{
		ID:        r.statusCounter,
		NodeID:    arg.NodeID,
		Submitted: arg.Submitted,
		Started:   arg.Started,
		Ended:     arg.Ended,
		Status:    arg.Status,
	}
	return nil
}

func (r *inMemDB) GetIOSpecByID(ctx context.Context, id int32) (IoSpec, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	i, ok := r.ioSpecs[id]
	if !ok {
		return IoSpec{}, ErrNotFound
	}
	return i, nil
}

func (r *inMemDB) GetNodeByID(ctx context.Context, id int32) (GetNodeByIDRow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.nodes[id]
	if !ok {
		return GetNodeByIDRow{}, ErrNotFound
	}
	var status Status
	for i := len(r.statuses); i > 0; i-- {
		s := r.statuses[int32(i)]
		if s.NodeID == n.ID {
			status = s
			break
		}
	}
	var result Result
	for i := len(r.results); i > 0; i-- {
		r := r.results[int32(i)]
		if r.NodeID == n.ID {
			result = r
			break
		}
	}
	var inputs []int32
	for _, i := range r.ioSpecs {
		if i.NodeID == n.ID && i.Type == "input" {
			inputs = append(inputs, i.ID)
		}
	}
	inputs = dedupAndSort(inputs)
	var outputs []int32
	for _, i := range r.ioSpecs {
		if i.NodeID == n.ID && i.Type == "output" {
			outputs = append(outputs, i.ID)
		}
	}
	outputs = dedupAndSort(outputs)
	var parents []int32
	for _, e := range r.edges {
		if e.ChildID == n.ID {
			parents = append(parents, e.ParentID)
		}
	}
	parents = dedupAndSort(parents)
	var children []int32
	for _, e := range r.edges {
		if e.ParentID == n.ID {
			children = append(children, e.ChildID)
		}
	}
	children = dedupAndSort(children)

	return GetNodeByIDRow{
		ID:          util.NullInt32(n.ID),
		QueueItemID: util.NullUUID(n.QueueItemID),
		Name:        util.NullStr(n.Name),
		Submitted:   status.Submitted,
		Started:     status.Started,
		Ended:       status.Ended,
		Status:      status.Status,
		ExecutionID: result.ExecutionID,
		Stdout:      result.Stdout,
		Stderr:      result.Stderr,
		Skipped:     result.Skipped,
		Inputs:      inputs,
		Outputs:     outputs,
		Parents:     parents,
		Children:    children,
	}, nil
}

func dedupAndSort(s []int32) []int32 {
	s = util.Dedup(s)
	util.SortSliceInt32(s)
	return s
}

func min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

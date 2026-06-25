package observability

import "time"

// TraceLink represents a single hop in a cross-agent distributed trace.
//
// RootTraceID identifies the entire trace tree. ParentTraceID is the span
// that triggered this hop, and ChildTraceID is this hop's own span ID.
// FromAgent and ToAgent identify the agents involved.
type TraceLink struct {
	RootTraceID   string    `json:"root_trace_id"`
	ParentTraceID string    `json:"parent_trace_id"`
	ChildTraceID  string    `json:"child_trace_id"`
	FromAgent     string    `json:"from_agent"`
	ToAgent       string    `json:"to_agent"`
	Action        string    `json:"action"`
	StartedAt     time.Time `json:"started_at"`
	DurationMs    int       `json:"duration_ms"`
}

// TraceTree represents a cross-agent trace as a tree structure built from a
// flat list of TraceLinks.
type TraceTree struct {
	Root     TraceLink    `json:"root"`
	Children []*TraceTree `json:"children"`
}

// BuildTraceTree builds a TraceTree from a flat slice of TraceLinks.
//
// The root link is identified as the one whose ParentTraceID is empty or
// equal to its RootTraceID. Remaining links are placed as children of the
// link whose ChildTraceID matches their ParentTraceID, recursively.
func BuildTraceTree(links []TraceLink) *TraceTree {
	if len(links) == 0 {
		return nil
	}

	// Build a set of all ChildTraceIDs to help identify the root.
	childIDs := make(map[string]bool, len(links))
	for _, l := range links {
		childIDs[l.ChildTraceID] = true
	}

	// The root link is the one whose ParentTraceID is not a ChildTraceID
	// of any other link (i.e. it has no parent in this set), or has an
	// empty ParentTraceID, or has ParentTraceID == RootTraceID.
	var root TraceLink
	rootIdx := -1
	for i, l := range links {
		if l.ParentTraceID == "" || l.ParentTraceID == l.RootTraceID {
			root = l
			rootIdx = i
			break
		}
		if !childIDs[l.ParentTraceID] {
			root = l
			rootIdx = i
			break
		}
	}

	if rootIdx == -1 {
		// Fallback: use the first link as root.
		root = links[0]
		rootIdx = 0
	}

	tree := &TraceTree{Root: root}
	tree.Children = buildChildren(root.ChildTraceID, links, rootIdx)
	return tree
}

func buildChildren(parentTraceID string, links []TraceLink, skipIdx int) []*TraceTree {
	var children []*TraceTree
	for i, l := range links {
		if i == skipIdx {
			continue
		}
		if l.ParentTraceID == parentTraceID {
			child := &TraceTree{Root: l}
			child.Children = buildChildren(l.ChildTraceID, links, -1)
			children = append(children, child)
		}
	}
	if len(children) == 0 {
		return nil
	}
	return children
}

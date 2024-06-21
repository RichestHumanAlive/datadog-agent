// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

// Package activitytree holds activitytree related files
package activitytree

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/hashicorp/golang-lru/v2/simplelru"
	"golang.org/x/sys/unix"

	"github.com/DataDog/datadog-agent/pkg/security/resolvers"
	"github.com/DataDog/datadog-agent/pkg/security/resolvers/process"
	"github.com/DataDog/datadog-agent/pkg/security/secl/model"
	"github.com/DataDog/datadog-agent/pkg/security/utils"
)

// NodeDroppedReason is used to list the reasons to drop a node
type NodeDroppedReason string

var (
	eventTypeReason       NodeDroppedReason = "event_type"
	invalidRootNodeReason NodeDroppedReason = "invalid_root_node"
	bindFamilyReason      NodeDroppedReason = "bind_family"
	brokenEventReason     NodeDroppedReason = "broken_event"
	allDropReasons                          = []NodeDroppedReason{
		eventTypeReason,
		invalidRootNodeReason,
		bindFamilyReason,
		brokenEventReason,
	}
)

var (
	// ErrBrokenLineage is returned when the given process don't have a full lineage
	ErrBrokenLineage = errors.New("broken lineage")
	// ErrNotValidRootNode is returned when trying to insert a process with an invalide root node
	ErrNotValidRootNode = errors.New("root node not valid")
)

// NodeGenerationType is used to indicate if a node was generated by a runtime or snapshot event
// IMPORTANT: IT MUST STAY IN SYNC WITH `adproto.GenerationType`
type NodeGenerationType byte

const (
	// Unknown is a node that was added at an unknown time
	Unknown NodeGenerationType = 0
	// Runtime is a node that was added at runtime
	Runtime NodeGenerationType = 1
	// Snapshot is a node that was added during the snapshot
	Snapshot NodeGenerationType = 2
	// ProfileDrift is a node that was added because of a drift from a security profile
	ProfileDrift NodeGenerationType = 3
	// WorkloadWarmup is a node that was added of a drift in a warming up profile
	WorkloadWarmup NodeGenerationType = 4
	// MaxNodeGenerationType is the maximum node type
	MaxNodeGenerationType NodeGenerationType = 4
)

func (genType NodeGenerationType) String() string {
	switch genType {
	case Runtime:
		return "runtime"
	case Snapshot:
		return "snapshot"
	case ProfileDrift:
		return "profile_drift"
	case WorkloadWarmup:
		return "workload_warmup"
	default:
		return "unknown"
	}
}

// Owner is used to communicate with the owner of the activity tree
type Owner interface {
	MatchesSelector(entry *model.ProcessCacheEntry) bool
	IsEventTypeValid(evtType model.EventType) bool
	NewProcessNodeCallback(p *ProcessNode)
}

type cookieSelector struct {
	execTime int64
	cookie   uint64
}

func (cs *cookieSelector) isSet() bool {
	return cs.execTime != 0 && cs.cookie != 0
}

func (cs *cookieSelector) fillFromEntry(entry *model.ProcessCacheEntry) {
	cs.execTime = entry.ExecTime.UnixNano()
	cs.cookie = entry.Cookie
}

// ActivityTree contains a process tree and its activities. This structure has no locks.
type ActivityTree struct {
	Stats *Stats

	treeType          string
	differentiateArgs bool
	DNSMatchMaxDepth  int

	validator    Owner
	pathsReducer *PathsReducer

	CookieToProcessNode *simplelru.LRU[cookieSelector, *ProcessNode]
	ProcessNodes        []*ProcessNode `json:"-"`

	// top level lists used to summarize the content of the tree
	DNSNames     *utils.StringKeys
	SyscallsMask map[int]int
}

// CookieToProcessNodeCacheSize defines the "cookie to process" node cache size
const CookieToProcessNodeCacheSize = 128

// NewActivityTree returns a new ActivityTree instance
func NewActivityTree(validator Owner, pathsReducer *PathsReducer, treeType string) *ActivityTree {
	cache, _ := simplelru.NewLRU[cookieSelector, *ProcessNode](CookieToProcessNodeCacheSize, nil)

	return &ActivityTree{
		treeType:            treeType,
		validator:           validator,
		pathsReducer:        pathsReducer,
		Stats:               NewActivityTreeNodeStats(),
		CookieToProcessNode: cache,
		SyscallsMask:        make(map[int]int),
		DNSNames:            utils.NewStringKeys(nil),
	}
}

// GetChildren returns the list of root ProcessNodes from the ActivityTree
func (at *ActivityTree) GetChildren() *[]*ProcessNode {
	return &at.ProcessNodes
}

// GetSiblings returns the list of siblings of the current node
func (at *ActivityTree) GetSiblings() *[]*ProcessNode {
	return nil
}

// AppendChild appends a new root node in the ActivityTree
func (at *ActivityTree) AppendChild(node *ProcessNode) {
	at.ProcessNodes = append(at.ProcessNodes, node)
	node.Parent = at
}

// AppendImageTag appends the given image tag
func (at *ActivityTree) AppendImageTag(_ string) {
}

// GetParent returns nil for the ActivityTree
func (at *ActivityTree) GetParent() ProcessNodeParent {
	return nil
}

// ComputeSyscallsList computes the top level list of syscalls
func (at *ActivityTree) ComputeSyscallsList() []uint32 {
	output := make([]uint32, 0, len(at.SyscallsMask))
	for key := range at.SyscallsMask {
		output = append(output, uint32(key))
	}
	sort.Slice(output, func(i, j int) bool {
		return output[i] < output[j]
	})
	return output
}

// ComputeActivityTreeStats computes the initial counts of the activity tree stats
func (at *ActivityTree) ComputeActivityTreeStats() {
	pnodes := at.ProcessNodes
	var fnodes []*FileNode

	for len(pnodes) > 0 {
		node := pnodes[0]

		at.Stats.ProcessNodes++
		pnodes = append(pnodes, node.Children...)

		at.Stats.DNSNodes += int64(len(node.DNSNames))
		at.Stats.SocketNodes += int64(len(node.Sockets))

		for _, f := range node.Files {
			fnodes = append(fnodes, f)
		}

		pnodes = pnodes[1:]
	}

	for len(fnodes) > 0 {
		node := fnodes[0]

		if node.File != nil {
			at.Stats.FileNodes++
		}

		for _, f := range node.Children {
			fnodes = append(fnodes, f)
		}

		fnodes = fnodes[1:]
	}
}

// IsEmpty returns true if the tree is empty
func (at *ActivityTree) IsEmpty() bool {
	return len(at.ProcessNodes) == 0
}

// Debug dumps the content of an activity tree
func (at *ActivityTree) Debug(w io.Writer) {
	for _, root := range at.ProcessNodes {
		root.debug(w, "")
	}
}

// ScrubProcessArgsEnvs scrubs and retains process args and envs
func (at *ActivityTree) ScrubProcessArgsEnvs(resolver *process.EBPFResolver) {
	// iterate through all the process nodes
	openList := make([]*ProcessNode, len(at.ProcessNodes))
	copy(openList, at.ProcessNodes)

	for len(openList) != 0 {
		current := openList[len(openList)-1]
		current.scrubAndReleaseArgsEnvs(resolver)
		openList = append(openList[:len(openList)-1], current.Children...)
	}
}

// DifferentiateArgs enables the args differentiation feature
func (at *ActivityTree) DifferentiateArgs() {
	at.differentiateArgs = true
}

type untracedEventError struct {
	eventType model.EventType
}

func (e untracedEventError) Error() string {
	return fmt.Sprintf("invalid event: event type not valid: %s", e.eventType)
}

// isEventValid evaluates if the provided event is valid
func (at *ActivityTree) isEventValid(event *model.Event, dryRun bool) (bool, error) {
	// check event type
	if !at.validator.IsEventTypeValid(event.GetEventType()) {
		if !dryRun {
			at.Stats.counts[event.GetEventType()].droppedCount[eventTypeReason].Inc()
		}
		return false, untracedEventError{eventType: event.GetEventType()}
	}

	// event specific filtering
	switch event.GetEventType() {
	case model.BindEventType:
		// ignore non IPv4 / IPv6 bind events for now
		if event.Bind.AddrFamily != unix.AF_INET && event.Bind.AddrFamily != unix.AF_INET6 {
			if !dryRun {
				at.Stats.counts[model.BindEventType].droppedCount[bindFamilyReason].Inc()
			}
			return false, errors.New("invalid event: invalid bind family")
		}
	case model.IMDSEventType:
		// ignore IMDS answers without AccessKeyIDS
		if event.IMDS.Type == model.IMDSResponseType && len(event.IMDS.AWS.SecurityCredentials.AccessKeyID) == 0 {
			return false, fmt.Errorf("untraced event: IMDS response without credentials")
		}
		// ignore IMDS requests without URLs
		if event.IMDS.Type == model.IMDSRequestType && len(event.IMDS.URL) == 0 {
			return false, fmt.Errorf("invalid event: IMDS request without any URL")
		}
	}
	return true, nil
}

// Insert inserts the event in the activity tree
func (at *ActivityTree) Insert(event *model.Event, insertMissingProcesses bool, imageTag string, generationType NodeGenerationType, resolvers *resolvers.EBPFResolvers) (bool, error) {
	newEntry, err := at.insertEvent(event, false /* !dryRun */, insertMissingProcesses, imageTag, generationType, resolvers)
	if newEntry {
		// this doesn't count the exec events which are counted separately
		at.Stats.counts[event.GetEventType()].addedCount[generationType].Inc()
	}
	return newEntry, err
}

// Contains looks up the event in the activity tree
func (at *ActivityTree) Contains(event *model.Event, insertMissingProcesses bool, imageTag string, generationType NodeGenerationType, resolvers *resolvers.EBPFResolvers) (bool, error) {
	newEntry, err := at.insertEvent(event, true /* dryRun */, insertMissingProcesses, imageTag, generationType, resolvers)
	return !newEntry, err
}

// insert inserts the event in the activity tree, returns true if the event generated a new entry in the tree
func (at *ActivityTree) insertEvent(event *model.Event, dryRun bool, insertMissingProcesses bool, imageTag string, generationType NodeGenerationType, resolvers *resolvers.EBPFResolvers) (bool, error) {
	// sanity check
	if generationType == Unknown || generationType > MaxNodeGenerationType {
		return false, fmt.Errorf("invalid generation type: %v", generationType)
	}

	// check if this event type is traced
	if valid, err := at.isEventValid(event, dryRun); !valid || err != nil {
		return false, err
	}

	// Next we'll call CreateProcessNode, which will retrieve the process node if already present, or create a new one (with all its lineage if needed).
	node, newProcessNode, err := at.CreateProcessNode(event.ProcessCacheEntry, imageTag, generationType, !insertMissingProcesses /*dryRun*/, resolvers)
	if err != nil {
		return false, err
	}
	if newProcessNode && !insertMissingProcesses {
		// the event insertion can't be done because there was missing process nodes for the related event we want to insert
		return true, nil
	} else if node == nil {
		// a process node couldn't be found or created for this event, ignore it
		return false, errors.New("a process node couldn't be found or created for this event")
	}

	// resolve fields
	event.ResolveFieldsForAD()

	// ignore events with an error
	if event.Error != nil {
		at.Stats.counts[event.GetEventType()].droppedCount[brokenEventReason].Inc()
		return false, event.Error
	}

	// the count of processed events is the count of events that matched the activity dump selector = the events for
	// which we successfully found a process activity node
	at.Stats.counts[event.GetEventType()].processedCount.Inc()

	// insert the event based on its type
	switch event.GetEventType() {
	case model.ExecEventType:
		// tag the matched rules if any
		node.MatchedRules = model.AppendMatchedRule(node.MatchedRules, event.Rules)
		return newProcessNode, nil
	case model.FileOpenEventType:
		return node.InsertFileEvent(&event.Open.File, event, imageTag, generationType, at.Stats, dryRun, at.pathsReducer, resolvers), nil
	case model.DNSEventType:
		return node.InsertDNSEvent(event, imageTag, generationType, at.Stats, at.DNSNames, dryRun, at.DNSMatchMaxDepth), nil
	case model.IMDSEventType:
		return node.InsertIMDSEvent(event, imageTag, generationType, at.Stats, dryRun), nil
	case model.BindEventType:
		return node.InsertBindEvent(event, imageTag, generationType, at.Stats, dryRun), nil
	case model.SyscallsEventType:
		return node.InsertSyscalls(event, at.SyscallsMask), nil
	case model.ExitEventType:
		// Update the exit time of the process (this is purely informative, do not rely on timestamps to detect
		// execed children)
		node.Process.ExitTime = event.Timestamp
	}

	return false, nil
}

func isContainerRuntimePrefix(basename string) bool {
	return strings.HasPrefix(basename, "runc") || strings.HasPrefix(basename, "containerd-shim")
}

// isValidRootNode evaluates if the provided process entry is allowed to become a root node of an Activity Dump
func isValidRootNode(entry *model.ProcessContext) bool {
	// an ancestor is required
	ancestor := GetNextAncestorBinaryOrArgv0(entry)
	if ancestor == nil {
		return false
	}

	if entry.FileEvent.IsFileless() {
		// a fileless node is a valid root node only if not having runc as parent
		// ex: runc -> exec(fileless) -> init.sh; exec(fileless) is not a valid root node
		return !(isContainerRuntimePrefix(ancestor.FileEvent.BasenameStr) || isContainerRuntimePrefix(entry.FileEvent.BasenameStr))
	}

	// container runtime prefixes are not valid root nodes
	return !isContainerRuntimePrefix(entry.FileEvent.BasenameStr)
}

// GetNextAncestorBinaryOrArgv0 returns the first ancestor with a different binary, or a different argv0 in the case of busybox processes
func GetNextAncestorBinaryOrArgv0(entry *model.ProcessContext) *model.ProcessCacheEntry {
	if entry == nil {
		return nil
	}
	current := entry
	ancestor := entry.Ancestor
	for ancestor != nil {
		if ancestor.FileEvent.Inode == 0 {
			return nil
		}
		if current.FileEvent.PathnameStr != ancestor.FileEvent.PathnameStr {
			return ancestor
		}
		if process.IsBusybox(current.FileEvent.PathnameStr) && process.IsBusybox(ancestor.FileEvent.PathnameStr) {
			currentArgv0, _ := process.GetProcessArgv0(&current.Process)
			if len(currentArgv0) == 0 {
				return nil
			}
			ancestorArgv0, _ := process.GetProcessArgv0(&ancestor.Process)
			if len(ancestorArgv0) == 0 {
				return nil
			}
			if currentArgv0 != ancestorArgv0 {
				return ancestor
			}
		}
		current = &ancestor.ProcessContext
		ancestor = ancestor.Ancestor
	}
	return nil
}

// buildBranchAndLookupCookies iterates over the ancestors of entry with 2 intentions in mind:
//   - check if one of the ancestors of entry is already in the tree and has a shortcut thanks to its cookie
//   - creates the list of ancestors "we care about" for the tree, i.e. the chain of ancestors created by calling
//     "GetNextAncestorBinaryOrArgv0" and that match the tree selector.
func (at *ActivityTree) buildBranchAndLookupCookies(entry *model.ProcessCacheEntry, imageTag string) ([]*model.ProcessCacheEntry, *ProcessNode, error) {
	var cs cookieSelector
	var fastMatch *ProcessNode
	var found bool
	var branch []*model.ProcessCacheEntry
	nextAncestor := entry

	for nextAncestor != nil {
		// look for the current ancestor
		cs.fillFromEntry(nextAncestor)
		if cs.isSet() {
			fastMatch, found = at.CookieToProcessNode.Get(cs)
			if found {
				fastMatch.applyImageTagOnLineageIfNeeded(imageTag)
				return branch, fastMatch, nil
			}
		}

		// check if the next ancestor matches the tree selector
		if !at.validator.MatchesSelector(nextAncestor) {
			// When the first ancestor that doesn't match the tree selector is reached, we can return early because we
			// know that none of its parents will match the selector
			break
		}

		// append current ancestor to the branch
		branch = append(branch, nextAncestor)
		nextAncestor = GetNextAncestorBinaryOrArgv0(&nextAncestor.ProcessContext)
	}
	if len(branch) == 0 {
		return nil, nil, nil
	}

	// make sure the branch has a valid root node
	for i := len(branch) - 1; i >= 0; i-- {
		if isValidRootNode(&branch[i].ProcessContext) {
			return branch[:i+1], nil, nil
		}
	}

	return branch, nil, ErrNotValidRootNode
}

// CreateProcessNode looks up or inserts the provided entry in the tree
func (at *ActivityTree) CreateProcessNode(entry *model.ProcessCacheEntry, imageTag string, generationType NodeGenerationType, dryRun bool, resolvers *resolvers.EBPFResolvers) (*ProcessNode, bool, error) {
	if entry == nil {
		return nil, false, nil
	}

	if _, err := entry.HasValidLineage(); err != nil {
		// check if the node belongs to the container
		var mn *model.ErrProcessMissingParentNode
		if !errors.As(err, &mn) {
			return nil, false, ErrBrokenLineage
		}
	}

	// Check if entry or one of its parents cookies are in CookieToProcessNode while building the branch we're trying to
	// insert.
	branchToInsert, quickMatch, err := at.buildBranchAndLookupCookies(entry, imageTag)
	if err != nil {
		return nil, false, err
	}
	if quickMatch != nil && len(branchToInsert) == 0 {
		// we can return immediately, we've found a direct hit from the cookie of the input entry
		return quickMatch, false, nil
	}

	var parent ProcessNodeParent

	// At this point, we want to insert "branchToInsert" below "firstMatch" in the tree
	if quickMatch == nil {
		// we didn't find a shortcut to the tree, this means we'll have to attempt the insertion from the top
		parent = at
	} else {
		// we have a shortcut to the tree, populate tree, siblings and parent accordingly
		parent = quickMatch
	}

	return at.insertBranch(parent, branchToInsert, imageTag, generationType, dryRun, resolvers)
}

func (at *ActivityTree) insertBranch(parent ProcessNodeParent, branchToInsert []*model.ProcessCacheEntry, imageTag string, generationType NodeGenerationType, dryRun bool, r *resolvers.EBPFResolvers) (*ProcessNode, bool, error) {
	var matchingNode *ProcessNode
	var branchIncrement int
	var newNode, newNodeFromRebase bool
	i := len(branchToInsert) - 1

	for i >= 0 {
		matchingNode, branchIncrement, newNodeFromRebase = at.findBranch(parent, branchToInsert[:i+1], dryRun, generationType, r)
		if newNodeFromRebase {
			newNode = true
			if dryRun {
				// early return in case of a dry run
				return nil, newNodeFromRebase, nil
			}
		}
		if matchingNode != nil {
			parent = matchingNode
			i -= branchIncrement
			continue
		}

		// we've found a new node in the branch which doesn't exist in the tree
		if dryRun {
			// exit early in case of a dry run
			return nil, true, nil
		}

		// we can safely insert the rest of the branch since they automatically all be new
		for j := i; j >= 0; j-- {
			// create the node
			matchingNode = NewProcessNode(branchToInsert[j], generationType, r)
			parent.AppendChild(matchingNode)

			// insert the new node in the list of children
			at.Stats.counts[model.ExecEventType].addedCount[generationType].Inc()
			at.Stats.ProcessNodes++

			parent = matchingNode
		}

		// if we reach this point, we can safely return the last inserted entry and indicate that the tree was modified
		matchingNode.applyImageTagOnLineageIfNeeded(imageTag)
		return matchingNode, true, nil
	}

	// if we reach this point, we've successfully found the matching node in the tree without modifying the tree
	if matchingNode != nil {
		matchingNode.applyImageTagOnLineageIfNeeded(imageTag)
	}
	return matchingNode, newNode, nil
}

// findBranch looks for the provided branch in the list of children. Returns the node that matches the
// first node of the branch and true if a new entry was inserted.
func (at *ActivityTree) findBranch(parent ProcessNodeParent, branch []*model.ProcessCacheEntry, dryRun bool, generationType NodeGenerationType, resolvers *resolvers.EBPFResolvers) (*ProcessNode, int, bool) {
	for i := len(branch) - 1; i >= 0; i-- {
		branchCursor := branch[i]

		// look for branchCursor in the children
		matchingNode, treeNodeToRebaseIndex := at.findProcessCacheEntryInTree(*parent.GetChildren(), branchCursor)

		if matchingNode != nil {
			// if this is the first iteration, we've just identified a direct match without looking for execs in the event.
			// This means we have nothing to rebase, return now.
			if i == len(branch)-1 {
				return matchingNode, 1, false
			}

			// we're about to rebase part of the tree, exit early if this is a dry run
			if dryRun {
				return nil, len(branch) - i, true
			}

			// make sure we properly update the IsExecExec status
			matchingNode.Process.IsExecExec = matchingNode.Process.IsExecExec || branchCursor.IsExecExec

			// here is the current state of the tree:
			//   parent -> treeNodeToRebase -> [...] -> matchingNode
			// here is what we want:
			//   parent -> { result of branch[i+1:].insert(treeNodeToRebase) } -> matchingNode
			at.rebaseTree(parent, treeNodeToRebaseIndex, parent, branch[i:], generationType, resolvers)

			return matchingNode, len(branch) - i, true
		}

		// are we looking for an exec child ?
		if siblings := parent.GetSiblings(); branchCursor.IsExecExec && siblings != nil {

			// if yes, then look for branchCursor in the siblings of the parent of children
			matchingNode, treeNodeToRebaseIndex = at.findProcessCacheEntryInTree(*siblings, branchCursor)
			if treeNodeToRebaseIndex >= 0 {

				// We're about to rebase part of the tree, exit early if this is a dry run.
				// The "i < len(branch)-1" check is used in case we'll rebase a node without adding a new one, which
				// should be allowed in a dryRun.
				if i < len(branch)-1 && dryRun {
					return nil, len(branch) - i, true
				}

				// make sure we properly update the IsExecExec status
				matchingNode.Process.IsExecExec = matchingNode.Process.IsExecExec || branchCursor.IsExecExec

				// here is the current state of the tree:
				//   parent of parent -> treeNodeToRebase -> [...] -> matchingNode
				// here is what we want:
				//   parent -> { result of branch[i+1:].insert(treeNodeToRebase) } -> matchingNode
				at.rebaseTree(parent.GetParent(), treeNodeToRebaseIndex, parent, branch[i:], generationType, resolvers)

				return matchingNode, len(branch) - i, i < len(branch)-1
			}
		}

		// We didn't find the current entry anywhere, has it execed into something else ? (i.e. are we missing something
		// in the profile ?)
		if i-1 >= 0 {
			if branch[i-1].IsExecExec {
				continue
			}
		}

		// if we're here, we've either reached the end of the list of children, or the next child wasn't
		// directly exec-ed
		break
	}
	return nil, 0, false
}

// rebaseTree rebases the node identified by "nodeIndexToRebase" in the input "tree" onto a newly created branch made of
// "branchToInsert" and appended to "treeToRebaseOnto". New nodes will be tagged with the input "generationType".
// This function returns the top level node, owner of the newly inserted branch that lead to the rebased node
func (at *ActivityTree) rebaseTree(parent ProcessNodeParent, childIndexToRebase int, newParent ProcessNodeParent, branchToInsert []*model.ProcessCacheEntry, generationType NodeGenerationType, resolvers *resolvers.EBPFResolvers) *ProcessNode {
	if len(branchToInsert) > 1 {
		// We know that the entries in "branch" are all "isExecChild = true" nodes, except the top level entry that might be
		// a "isExecChild = false" node. Similarly, all the nodes below parent.GetChildren()[childIndexToRebase] must be non
		// matching "isExecChild = true" nodes, except parent.GetChildren()[childIndexToRebase] that might be a "isExecChild
		// = false" node. To be safe, check if the 2 top level nodes match if one of them is an "isExecChild = true" node.
		childToRebase := (*parent.GetChildren())[childIndexToRebase]
		if topLevelNode := branchToInsert[len(branchToInsert)-1]; !topLevelNode.IsExecExec || !childToRebase.Process.IsExecExec {
			if childToRebase.Matches(&topLevelNode.Process, at.differentiateArgs, true) {
				// ChildNodeToRebase and topLevelNode match and need to be merged, rebase the one in the profile, and insert
				// the remaining nodes of the branch on top of it
				newRebasedChild := at.rebaseTree(parent, childIndexToRebase, newParent, nil, generationType, resolvers)
				output, _, _ := at.insertBranch(newRebasedChild, branchToInsert[:len(branchToInsert)-1], "", generationType, false, resolvers)

				if output == nil {
					return newRebasedChild
				}
				return output
			}
		}
	}

	// create the new branch
	var rebaseRoot, childrenCursor *ProcessNode
	for i := len(branchToInsert) - 1; i >= 1; i-- {
		eventExecChildTmp := branchToInsert[i]
		n := NewProcessNode(eventExecChildTmp, generationType, resolvers)
		if i == len(branchToInsert)-1 {
			rebaseRoot = n
		}
		if childrenCursor != nil {
			childrenCursor.AppendChild(n)
		}
		at.Stats.ProcessNodes++
		at.Stats.counts[model.ExecEventType].addedCount[generationType].Inc()

		childrenCursor = n
	}

	// mark the rebased node as an exec child
	(*parent.GetChildren())[childIndexToRebase].Process.IsExecExec = true

	if rebaseRoot == nil {
		rebaseRoot = (*parent.GetChildren())[childIndexToRebase]
	}

	if childrenCursor != nil {
		// attach the head of to the last newly inserted child
		childrenCursor.Children = append(childrenCursor.Children, (*parent.GetChildren())[childIndexToRebase])
	}

	// rebase the node onto treeToRebaseOnto
	*newParent.GetChildren() = append(*newParent.GetChildren(), rebaseRoot)

	// break the link between the parent and the node to rebase
	*parent.GetChildren() = append((*parent.GetChildren())[0:childIndexToRebase], (*parent.GetChildren())[childIndexToRebase+1:]...)

	// now that the tree is ready, call the validator on the first node
	at.validator.NewProcessNodeCallback(rebaseRoot)

	return rebaseRoot
}

// findProcessCacheEntryInTree looks for the provided entry in the list of process nodes, returns the node (if
// found) and the index of the top level child that lead to the matching node (or -1 if not found).
func (at *ActivityTree) findProcessCacheEntryInTree(tree []*ProcessNode, entry *model.ProcessCacheEntry) (*ProcessNode, int) {
	for i, child := range tree {
		if child.Matches(&entry.Process, at.differentiateArgs, true) {
			return child, i
		}
	}

	for i, child := range tree {
		// has the parent execed into one of its own children ?
		if execChild := at.findProcessCacheEntryInChildExecedNodes(child, entry); execChild != nil {
			return execChild, i
		}
	}
	return nil, -1
}

// findProcessCacheEntryInChildExecedNodes look for entry in the execed nodes of child
func (at *ActivityTree) findProcessCacheEntryInChildExecedNodes(child *ProcessNode, entry *model.ProcessCacheEntry) *ProcessNode {
	// fast path
	for _, node := range child.Children {
		if node.Process.IsExecExec {
			// does this execed child match the entry ?
			if node.Matches(&entry.Process, at.differentiateArgs, true) {
				return node
			}
		}
	}

	// slow path

	// children is used to iterate over the tree below child
	execChildren := make([]*ProcessNode, 1, 64)
	execChildren[0] = child

	visited := make([]*ProcessNode, 0, 64)

	for len(execChildren) > 0 {
		cursor := execChildren[len(execChildren)-1]
		execChildren = execChildren[:len(execChildren)-1]

		visited = append(visited, cursor)

		// look for an execed child
		for _, node := range cursor.Children {
			if node.Process.IsExecExec && !slices.Contains(visited, node) {
				// there should always be only one

				// does this execed child match the entry ?
				if node.Matches(&entry.Process, at.differentiateArgs, true) {
					return node
				}

				execChildren = append(execChildren, node)
			}
		}
	}

	// not found
	return nil
}

// FindMatchingRootNodes finds and returns the matching root nodes
func (at *ActivityTree) FindMatchingRootNodes(arg0 string) []*ProcessNode {
	var res []*ProcessNode
	for _, node := range at.ProcessNodes {
		if node.Process.Argv0 == arg0 {
			res = append(res, node)
		}
	}
	return res
}

// Snapshot uses procfs to snapshot the nodes of the tree
func (at *ActivityTree) Snapshot(newEvent func() *model.Event) {
	for _, pn := range at.ProcessNodes {
		pn.snapshot(at.validator, at.Stats, newEvent, at.pathsReducer)
	}
}

// SendStats sends the tree statistics
func (at *ActivityTree) SendStats(client statsd.ClientInterface) error {
	return at.Stats.SendStats(client, at.treeType)
}

// TagAllNodes tags all the activity tree's nodes with the given image tag
func (at *ActivityTree) TagAllNodes(imageTag string) {
	for _, rootNode := range at.ProcessNodes {
		rootNode.TagAllNodes(imageTag)
	}
}

// EvictImageTag will remove every trace of the given image tag from the tree
func (at *ActivityTree) EvictImageTag(imageTag string) {
	// purge the cookies which todays are never set. TODO: once they'll get used, recompute them here
	at.CookieToProcessNode.Purge()

	// recompute also the full list of DNSNames and Syscalls when evicting nodes
	DNSNames := utils.NewStringKeys(nil)
	SyscallsMask := make(map[int]int)
	newProcessNodes := []*ProcessNode{}
	for _, node := range at.ProcessNodes {
		if shouldRemoveNode := node.EvictImageTag(imageTag, DNSNames, SyscallsMask); !shouldRemoveNode {
			newProcessNodes = append(newProcessNodes, node)
		}
	}
	at.ProcessNodes = newProcessNodes
}
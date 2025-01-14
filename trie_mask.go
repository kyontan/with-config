package main

import (
	"strings"
)

type TrieNode struct {
	children map[rune]*TrieNode
	position int
	isEnd    bool
}

type MatchedRange struct {
	start  int
	length int
}

type TrieNodeState struct {
	nodes         []*TrieNode
	matchedRanges []MatchedRange
}

func NewTrieTree() *TrieNode {
	return NewTrieNode(0)
}

func NewTrieNode(position int) *TrieNode {
	return &TrieNode{children: make(map[rune]*TrieNode), position: position}
}

func NewTrieNodeState() *TrieNodeState {
	return &TrieNodeState{
		nodes:         []*TrieNode{},
		matchedRanges: []MatchedRange{},
	}
}

func (node *TrieNode) Insert(word string) {
	current := node
	for _, ch := range word {
		if _, exists := current.children[ch]; !exists {
			current.children[ch] = NewTrieNode(current.position + 1)
		}
		current = current.children[ch]
	}
	current.isEnd = true
}

func (node *TrieNode) step(c rune, runePosition int, state *TrieNodeState) *TrieNodeState {
	var newState *TrieNodeState
	if state == nil {
		newState = NewTrieNodeState()
	} else {
		newState = state
	}

	// add root to new state
	newState.nodes = append(newState.nodes, node)

	newCurrentNodes := []*TrieNode{}
	for _, current := range newState.nodes {
		if next, exists := current.children[c]; exists {
			if next.isEnd { // match found
				newState.matchedRanges = append(newState.matchedRanges, MatchedRange{runePosition - next.position + 1, next.position})
			}

			// if next have children, add it to newCurrentNodes
			if len(next.children) > 0 {
				newCurrentNodes = append(newCurrentNodes, next)
			}
		}

		// special handling for backslack ("\\") character: keep (append) the last matching node (with new position)
		if c == '\\' {
			newCurrentNodes = append(newCurrentNodes, current)
		}
	}

	newState.nodes = newCurrentNodes

	return newState
}

func isRangesContainAt(ranges []MatchedRange, i int) bool {
	for _, r := range ranges {
		if r.start <= i && i < (r.start+r.length) {
			return true
		}
	}
	return false
}

func (node *TrieNode) Mask(text string, state *TrieNodeState) (masked string, matching string, newState *TrieNodeState) {
	var result strings.Builder

	currentState := NewTrieNodeState()

	currentState.nodes = append(currentState.nodes, state.nodes...)
	currentState.matchedRanges = append(currentState.matchedRanges, state.matchedRanges...)

	// if startNodes is empty, start from root
	if len(currentState.nodes) == 0 {
		currentState.nodes = append(currentState.nodes, node)
	}

	printedPos := 0

	for i, ch := range text {
		currentState = node.step(ch, i, currentState)

		// if there is no matching node: print all the characters from printedPos to i
		// if there is only newly added nodes, print all the characters from printedPos to i - 1
		// otherwise, do nothing (keep matching)
		allNodesAreNewlyAdded := true
		for _, n := range currentState.nodes {
			if n.position > 1 {
				allNodesAreNewlyAdded = false
				break
			}
		}

		// if no current matching node found, we can print the character
		if allNodesAreNewlyAdded {

			end := i
			if len(currentState.nodes) > 0 { // if there is under matching, we cannot print the pos i
				end--
			}

			for printedPos <= end {
				if isRangesContainAt(currentState.matchedRanges, printedPos) {
					result.WriteString("*")
				} else {
					result.WriteRune(rune(text[printedPos]))
				}

				printedPos++
			}

			// keeps only the ranges that still active
			newMatchedRanges := []MatchedRange{}
			for _, r := range currentState.matchedRanges {
				if r.start+r.length > printedPos {
					newMatchedRanges = append(newMatchedRanges, r)
				}
			}

			currentState.matchedRanges = newMatchedRanges
		}
	}

	// shift start of matchedRanges
	for i := range currentState.matchedRanges {
		currentState.matchedRanges[i].start -= printedPos
	}

	return result.String(), text[printedPos:], currentState
}

func (node *TrieNode) PrintRemaining(text string, state *TrieNodeState) string {
	var result strings.Builder

	for i, ch := range text {
		if isRangesContainAt(state.matchedRanges, i) {
			result.WriteString("*")
		} else if len(state.nodes) != 0 {
			// maybe the character is a part of a matching secret
			result.WriteString("*")
		} else {
			result.WriteRune(ch)
		}
	}

	return result.String()
}

func BuildTrieFromSecrets(secrets []string) *TrieNode {
	root := NewTrieTree()
	for _, secret := range secrets {
		root.Insert(secret)
	}
	return root
}

func BuildTrieFromSecretsMap(secrets map[string]string) *TrieNode {
	root := NewTrieTree()
	for _, secret := range secrets {
		root.Insert(secret)
	}
	return root
}

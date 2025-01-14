package main

import (
	"fmt"
	"os"
	"strings"
)

type TrieNode struct {
	children map[rune]*TrieNode
	position int
	isEnd    bool
}

type MatchRange struct {
	start  int
	length int
	node   *TrieNode // if not nil, still matching, otherwise (nil): already matched
}

type TrieNodeState struct {
	matchRanges []MatchRange
}

func NewTrieTree() *TrieNode {
	return NewTrieNode(0)
}

func NewTrieNode(position int) *TrieNode {
	return &TrieNode{children: make(map[rune]*TrieNode), position: position}
}

func NewTrieNodeState() *TrieNodeState {
	return &TrieNodeState{
		matchRanges: []MatchRange{},
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
	// clone states
	currentState := NewTrieNodeState()
	currentState.matchRanges = append(currentState.matchRanges, state.matchRanges...)

	// add root node to current state (match from the beginning)
	currentState.matchRanges = append(currentState.matchRanges, MatchRange{
		start:  runePosition,
		length: 0,
		node:   node,
	})

	newState := NewTrieNodeState()

	for _, current := range currentState.matchRanges {
		currentNode := current.node

		// if the current node is nil, keep it
		if currentNode == nil {
			newState.matchRanges = append(newState.matchRanges, current)
			continue
		}

		if next, exists := currentNode.children[c]; exists {
			if next.isEnd { // match found
				newState.matchRanges = append(newState.matchRanges, MatchRange{
					start:  current.start,
					length: current.length + 1,
					node:   nil, // means end of the match
				})
			}

			// if next have children, add it to newCurrentNodes
			if len(next.children) > 0 {
				newState.matchRanges = append(newState.matchRanges, MatchRange{
					start:  current.start,
					length: current.length + 1,
					node:   next,
				})
			}
		} else {
			// special handling for backslack ("\\") character: keep (append) the last matching node (with new position)
			if c == '\\' {
				newState.matchRanges = append(newState.matchRanges, MatchRange{
					start:  current.start,
					length: current.length + 1,
					node:   current.node,
				})
			}
		}
	}

	return newState
}

// only considers the ranges that have node == nil (already matched)
func isRangesContainAt(ranges []MatchRange, i int) bool {
	for _, r := range ranges {
		if r.node == nil && r.start <= i && i < (r.start+r.length) {
			return true
		}
	}
	return false
}

func DebugPrintState(state *TrieNodeState, text string) {
	if os.Getenv("DEBUG") != "true" {
		return
	}
	for i, r := range state.matchRanges {
		if i > 10 {
			return
		}

		nodeStr := ""
		if r.node == nil {
			nodeStr = "nil"
		} else {
			nodeStr = "non-nil"
		}

		var startChar rune
		var endChar rune
		if r.start < 0 {
			startChar = '?'
		} else {
			startChar = rune(text[r.start])
		}

		if r.start+r.length-1 < 0 {
			endChar = '?'
		} else {
			endChar = rune(text[r.start+r.length-1])
		}

		fmt.Printf("i: %d, mr: [%c .. %c] (start: %d, len: %d), node: %s\n", i, startChar, endChar, r.start, r.length, nodeStr)
	}

	if len(state.matchRanges) == 0 {
		fmt.Printf("No match\n")
	}
}

func (node *TrieNode) Mask(text string, state *TrieNodeState) (masked string, matching string, newState *TrieNodeState) {
	var result strings.Builder

	currentState := NewTrieNodeState()

	currentState.matchRanges = append(currentState.matchRanges, state.matchRanges...)

	printedPos := 0

	for i, ch := range text {
		if os.Getenv("DEBUG") == "true" {
			fmt.Print("Before step: ")
			DebugPrintState(currentState, text)
		}

		currentState = node.step(ch, i, currentState)

		if os.Getenv("DEBUG") == "true" {
			fmt.Print("After step : ")
			DebugPrintState(currentState, text)
		}

		// if there is no matching node: print all the characters from printedPos to i
		// if there is only newly added nodes, print all the characters from printedPos to i - 1
		// otherwise, do nothing (keep matching)
		numNewMatchingNodes := 0
		numExistingMatchingNodes := 0
		for _, mr := range currentState.matchRanges {
			if mr.node != nil {
				if mr.node.position == 0 {
					numNewMatchingNodes++
				} else {
					numExistingMatchingNodes++
				}
			}
		}

		if numExistingMatchingNodes == 0 {
			end := i
			if numNewMatchingNodes > 0 { // if there is under matching, we cannot print the pos i
				end--
			}

			for printedPos <= end {
				if isRangesContainAt(currentState.matchRanges, printedPos) {
					result.WriteString("*")
				} else {
					result.WriteRune(rune(text[printedPos]))
				}

				printedPos++
			}

			// keeps only the ranges that still active
			newMatchRanges := []MatchRange{}
			for _, r := range currentState.matchRanges {
				if r.start+r.length > printedPos {
					newMatchRanges = append(newMatchRanges, r)
				}
			}

			currentState.matchRanges = newMatchRanges
		}
	}

	// shift start of matchRanges
	for i := range currentState.matchRanges {
		currentState.matchRanges[i].start -= printedPos
	}

	return result.String(), text[printedPos:], currentState
}

func (node *TrieNode) PrintRemaining(text string, state *TrieNodeState) string {
	var result strings.Builder

	for i, ch := range text {
		if isRangesContainAt(state.matchRanges, i) {
			result.WriteString("*")
		} else if len(state.matchRanges) != 0 {
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

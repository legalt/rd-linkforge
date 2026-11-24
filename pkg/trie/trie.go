// Package trie implements a byte-wise prefix tree for strings.
package trie

type node struct {
	children [256]*node
	word     bool
}

type Trie struct {
	root node
}

func New() *Trie {
	return &Trie{}
}

// Insert adds word unless it is empty or already present.
func (t *Trie) Insert(word string) bool {
	if word == "" {
		return false
	}

	n := &t.root
	for i := 0; i < len(word); i++ {
		b := word[i]
		if n.children[b] == nil {
			n.children[b] = &node{}
		}
		n = n.children[b]
	}
	if n.word {
		return false
	}

	n.word = true
	return true
}

func (t *Trie) Contains(word string) bool {
	n := t.find(word)
	return word != "" && n != nil && n.word
}

// Delete removes word and prunes nodes that no longer lead to a word.
func (t *Trie) Delete(word string) bool {
	if word == "" {
		return false
	}

	n := &t.root
	path := make([]*node, len(word)+1)
	path[0] = n
	for i := 0; i < len(word); i++ {
		n = n.children[word[i]]
		if n == nil {
			return false
		}
		path[i+1] = n
	}
	if !n.word {
		return false
	}

	n.word = false
	for i := len(word); i > 0; i-- {
		if path[i].word || hasChildren(path[i]) {
			break
		}
		path[i-1].children[word[i-1]] = nil
	}
	return true
}

// WithPrefix returns matching words in ascending byte order.
func (t *Trie) WithPrefix(prefix string, limit int) []string {
	n := t.find(prefix)
	if n == nil {
		return []string{}
	}

	words := make([]string, 0)
	path := []byte(prefix)
	var visit func(*node) bool
	visit = func(current *node) bool {
		if current.word {
			words = append(words, string(path))
			if limit > 0 && len(words) == limit {
				return true
			}
		}
		for i, child := range current.children {
			if child == nil {
				continue
			}
			path = append(path, byte(i))
			stop := visit(child)
			path = path[:len(path)-1]
			if stop {
				return true
			}
		}
		return false
	}
	visit(n)
	return words
}

func (t *Trie) find(value string) *node {
	n := &t.root
	for i := 0; i < len(value); i++ {
		n = n.children[value[i]]
		if n == nil {
			return nil
		}
	}
	return n
}

func hasChildren(n *node) bool {
	for _, child := range n.children {
		if child != nil {
			return true
		}
	}
	return false
}

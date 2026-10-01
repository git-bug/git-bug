package repository

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/index/upsidedown"
)

var _ Index = &bleveIndex{}

type bleveIndex struct {
	path string

	mu    sync.RWMutex
	index bleve.Index
}

func openBleveIndex(path string) (*bleveIndex, error) {
	index, err := bleve.Open(path)
	if err != nil {
		// likely we have no index yet, we make one.
		b := &bleveIndex{path: path}
		return b, b.makeIndex()
	}

	adv, err := index.Advanced()
	if err != nil {
		_ = index.Close()
		return nil, fmt.Errorf("bleve: couldn't get the advanced index to assert index type: %v", err)
	}

	// if we detect the v1 format (upside-down), we force a rebuild to the v2 format (scorch)
	// which is much smaller.
	if _, ok := adv.(*upsidedown.UpsideDownCouch); ok {
		_ = index.Close()
		err = os.RemoveAll(path)
		if err != nil {
			return nil, err
		}
		b := &bleveIndex{path: path}
		return b, b.makeIndex()
	}

	return &bleveIndex{path: path, index: index}, nil
}

func (b *bleveIndex) makeIndex() error {
	err := os.MkdirAll(b.path, os.ModePerm)
	if err != nil {
		return err
	}

	// TODO: follow https://github.com/blevesearch/bleve/issues/1576 recommendations

	mapping := bleve.NewIndexMapping()
	mapping.DefaultAnalyzer = "en"

	index, err := bleve.New(b.path, mapping)
	if err != nil {
		return err
	}
	b.index = index
	return nil
}

// builtFromKey is the internal value of the index holding, for each document,
// the commit it was built from. Internal values are applied and persisted with
// the documents of the same batch, so a document and its commit land together.
var builtFromKey = []byte("git-bug:built-from")

func (b *bleveIndex) NewBatch() IndexBatch {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return &bleveBatch{
		index:   b,
		batch:   b.index.NewBatch(),
		changes: make(map[string]Hash),
	}
}

func (b *bleveIndex) BuiltFrom() (map[string]Hash, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.builtFrom()
}

// builtFrom reads the record of what the documents were built from. No value at
// all is an empty record: a new or cleared index, or one written before the
// index recorded anything. A record that can't be decoded fails, and has the
// cache rebuild the index, which also upgrades a record in an older format.
func (b *bleveIndex) builtFrom() (map[string]Hash, error) {
	raw, err := b.index.GetInternal(builtFromKey)
	if err != nil {
		return nil, err
	}
	builtFrom := make(map[string]Hash)
	if len(raw) == 0 {
		return builtFrom, nil
	}
	err = gob.NewDecoder(bytes.NewReader(raw)).Decode(&builtFrom)
	if err != nil {
		return nil, fmt.Errorf("malformed index record: %w", err)
	}
	return builtFrom, nil
}

type bleveBatch struct {
	index *bleveIndex
	batch *bleve.Batch
	// the commit of each document set, or "" for a removed one
	changes map[string]Hash
}

func (bb *bleveBatch) Set(id string, texts []string, commit Hash) error {
	if !commit.IsValid() {
		return fmt.Errorf("invalid commit %q for document %s", commit, id)
	}

	// drop the very long words, see https://github.com/blevesearch/bleve/issues/1576
	normalized := make([]string, len(texts))
	var sb strings.Builder
	for i, text := range texts {
		sb.Reset()
		for _, field := range strings.Fields(text) {
			if utf8.RuneCountInString(field) < 100 {
				sb.WriteString(field)
				sb.WriteRune(' ')
			}
		}
		normalized[i] = sb.String()
	}

	err := bb.batch.Index(id, struct{ Text []string }{Text: normalized})
	if err != nil {
		return err
	}
	bb.changes[id] = commit
	return nil
}

func (bb *bleveBatch) Remove(id string) {
	bb.batch.Delete(id)
	bb.changes[id] = ""
}

func (bb *bleveBatch) Apply() error {
	b := bb.index
	b.mu.Lock()
	defer b.mu.Unlock()

	// under the lock, so that two batches don't both update the same record
	builtFrom, err := b.builtFrom()
	if err != nil {
		return err
	}
	for id, commit := range bb.changes {
		if commit == "" {
			delete(builtFrom, id)
		} else {
			builtFrom[id] = commit
		}
	}
	var raw bytes.Buffer
	err = gob.NewEncoder(&raw).Encode(builtFrom)
	if err != nil {
		return err
	}
	bb.batch.SetInternal(builtFromKey, raw.Bytes())

	return b.index.Batch(bb.batch)
}

func (b *bleveIndex) Search(terms []string) ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for i, term := range terms {
		if strings.Contains(term, " ") {
			terms[i] = fmt.Sprintf("\"%s\"", term)
		}
	}

	query := bleve.NewQueryStringQuery(strings.Join(terms, " "))
	search := bleve.NewSearchRequest(query)

	res, err := b.index.Search(search)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(res.Hits))
	for i, hit := range res.Hits {
		ids[i] = hit.ID
	}

	return ids, nil
}

func (b *bleveIndex) Clear() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	err := b.index.Close()
	if err != nil {
		return err
	}

	err = os.RemoveAll(b.path)
	if err != nil {
		return err
	}

	return b.makeIndex()
}

func (b *bleveIndex) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.index.Close()
}

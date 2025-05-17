package repository

import (
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

func (b *bleveIndex) IndexOne(id string, texts []string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b._index(b.index.Index, id, texts)
}

func (b *bleveIndex) IndexBatch() (indexer func(id string, texts []string) error, closer func() error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	batch := b.index.NewBatch()

	indexer = func(id string, texts []string) error {
		return b._index(batch.Index, id, texts)
	}

	closer = func() error {
		return b.index.Batch(batch)
	}

	return indexer, closer
}

func (b *bleveIndex) _index(indexer func(string, interface{}) error, id string, texts []string) error {
	searchable := struct{ Text []string }{Text: texts}

	// See https://github.com/blevesearch/bleve/issues/1576
	var sb strings.Builder
	normalize := func(text string) string {
		sb.Reset()
		for _, field := range strings.Fields(text) {
			if utf8.RuneCountInString(field) < 100 {
				sb.WriteString(field)
				sb.WriteRune(' ')
			}
		}
		return sb.String()
	}

	for i, s := range searchable.Text {
		searchable.Text[i] = normalize(s)
	}

	return indexer(id, searchable)
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

func (b *bleveIndex) DocCount() (uint64, error) {
	return b.index.DocCount()
}

func (b *bleveIndex) Remove(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.index.Delete(id)
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

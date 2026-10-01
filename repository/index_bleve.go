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
	"github.com/blevesearch/bleve/v2/analysis/lang/en"
	"github.com/blevesearch/bleve/v2/index/upsidedown"
	"github.com/blevesearch/bleve/v2/search/query"
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

	// An index in the v1 format (upside-down), or with another layout, is
	// replaced by an empty one. The cache then sees an index that doesn't match
	// its excerpts, and rebuilds both.
	_, upsideDown := adv.(*upsidedown.UpsideDownCouch)
	version, err := index.GetInternal(layoutVersionKey)
	if err != nil {
		_ = index.Close()
		return nil, err
	}
	if upsideDown || !bytes.Equal(version, layoutVersion) {
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

// layoutVersionKey is the internal value of the index holding the version of
// its layout: the mapping, and how the documents are written. layoutVersion
// must change with them, so that existing indexes get rebuilt.
var layoutVersionKey = []byte("git-bug:layout")
var layoutVersion = []byte("1")

func (b *bleveIndex) makeIndex() error {
	err := os.MkdirAll(b.path, os.ModePerm)
	if err != nil {
		return err
	}

	// Only the ids come out of a search, so the text is indexed but neither
	// stored nor kept for sorting, and not copied in the _all field either.
	// The positions of the terms are kept, for phrase searches.
	// See https://github.com/blevesearch/bleve/issues/1576
	text := bleve.NewTextFieldMapping()
	text.Analyzer = en.AnalyzerName
	text.Store = false
	text.DocValues = false
	text.IncludeInAll = false
	text.IncludeTermVectors = true

	doc := bleve.NewDocumentStaticMapping()
	doc.AddFieldMappingsAt(textField, text)

	mapping := bleve.NewIndexMapping()
	mapping.DefaultMapping = doc
	mapping.DefaultAnalyzer = en.AnalyzerName
	mapping.DefaultField = textField
	mapping.IndexDynamic = false
	mapping.StoreDynamic = false
	mapping.DocValuesDynamic = false

	index, err := bleve.New(b.path, mapping)
	if err != nil {
		return err
	}
	err = index.SetInternal(layoutVersionKey, layoutVersion)
	if err != nil {
		_ = index.Close()
		return err
	}
	b.index = index
	return nil
}

// textField is the field of the documents holding their text.
const textField = "Text"

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
	var normalized []string
	var sb strings.Builder
	for _, text := range texts {
		sb.Reset()
		for _, field := range strings.Fields(text) {
			if utf8.RuneCountInString(field) < 100 {
				sb.WriteString(field)
				sb.WriteRune(' ')
			}
		}
		if sb.Len() > 0 {
			normalized = append(normalized, sb.String())
		}
	}

	// a document without text can't match a search, it's only recorded
	if len(normalized) == 0 {
		bb.batch.Delete(id)
	} else {
		err := bb.batch.Index(id, struct{ Text []string }{Text: normalized})
		if err != nil {
			return err
		}
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

	// a document matches if it has any of the terms. A term is a phrase: a
	// quoted text, or a word the analyzer splits into several (foo-bar, file.go,
	// a word in a language written without spaces).
	// The terms are matched as text, never as a query syntax.
	termQueries := make([]query.Query, len(terms))
	for i, term := range terms {
		q := bleve.NewMatchPhraseQuery(term)
		q.SetField(textField)
		termQueries[i] = q
	}

	// every match, rather than bleve's default of the 10 best
	count, err := b.index.DocCount()
	if err != nil {
		return nil, err
	}

	search := bleve.NewSearchRequestOptions(bleve.NewDisjunctionQuery(termQueries...), int(count), 0, false)
	// the matches are returned as a set, their relevance is of no use
	search.Score = bleve.ScoreNone

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

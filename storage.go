package watertower

import (
	"context"
	"encoding/gob"
	"errors"
	"io"
	"sync"

	"github.com/shibukawa/cloudcounter"
	"gocloud.dev/docstore"
)

const (
	documentID    cloudcounter.CounterKey = "document_id"
	documentCount                         = "document_count"
)

func init() {
	gob.Register(&Document{})
	gob.Register(&documentKey{})
	gob.Register(&tokenEntity{})
	gob.Register(&tagEntity{})
}

type Storage interface {
	IncrementDocID() (int, error)
	IncrementDocCount() (int, error)
	DecrementDocCount() error
	DocCount() (int, error)
	Context() context.Context
	Create(id string, doc interface{}) error
	Replace(id string, doc interface{}) error
	GetDoc(doc *Document) error
	GetDocKey(docKey *documentKey) error
	GetTag(tag *tagEntity) error
	GetToken(token *tokenEntity) error
	BatchDocGet(ctx context.Context, docs []*Document) (map[int]bool, error)
	BatchTagGet(ctx context.Context, tags []*tagEntity) (map[int]bool, error)
	BatchTokenGet(ctx context.Context, tokens []*tokenEntity) (map[int]bool, error)
	DeleteDoc(doc *Document) error
	DeleteDocKey(docKey *documentKey) error
	DeleteTag(tag *tagEntity) error
	DeleteToken(token *tokenEntity) error
	Close() error
	WriteIndex(w io.Writer) error
	ReadIndex(r io.Reader) error
}

var (
	_ Storage = &docstoreStorage{}
	_ Storage = &localStorage{}
)

type docstoreStorage struct {
	ctx        context.Context
	collection *docstore.Collection
	counter    *cloudcounter.Counter
}

func (d *docstoreStorage) DocCount() (int, error) {
	return d.counter.Get(d.ctx, documentCount)
}

func (d *docstoreStorage) WriteIndex(w io.Writer) error {
	return nil
}

func (d *docstoreStorage) ReadIndex(r io.Reader) error {
	return nil
}

func (d *docstoreStorage) Context() context.Context {
	return d.ctx
}

func (d *docstoreStorage) IncrementDocID() (int, error) {
	return d.counter.Increment(d.ctx, documentID)
}

func (d *docstoreStorage) IncrementDocCount() (int, error) {
	return d.counter.Increment(d.ctx, documentCount)
}

func (d *docstoreStorage) DecrementDocCount() error {
	return d.counter.Decrement(d.ctx, documentCount)
}

func (d *docstoreStorage) Create(id string, doc interface{}) error {
	return d.collection.Create(d.ctx, doc)
}

func (d *docstoreStorage) Replace(id string, doc interface{}) error {
	return d.collection.Replace(d.ctx, doc)
}

func (d *docstoreStorage) GetDoc(doc *Document) error {
	return d.collection.Get(d.ctx, doc)
}

func (d *docstoreStorage) GetDocKey(docKey *documentKey) error {
	return d.collection.Get(d.ctx, docKey)
}

func (d *docstoreStorage) GetTag(tag *tagEntity) error {
	return d.collection.Get(d.ctx, tag)
}

func (d *docstoreStorage) GetToken(token *tokenEntity) error {
	return d.collection.Get(d.ctx, token)
}

func (d *docstoreStorage) BatchDocGet(ctx context.Context, docs []*Document) (errs map[int]bool, err error) {
	actions := d.collection.Actions()
	for i := range docs {
		actions = actions.Get(docs[i])
	}
	err = actions.Do(ctx)
	if errs, ok := err.(docstore.ActionListError); ok {
		hasErrors := make(map[int]bool)
		for _, err := range errs {
			hasErrors[err.Index] = true
		}
		return hasErrors, err
	}
	return nil, err
}

func (d *docstoreStorage) BatchTagGet(ctx context.Context, tags []*tagEntity) (errs map[int]bool, err error) {
	actions := d.collection.Actions()
	for i := range tags {
		actions = actions.Get(tags[i])
	}
	err = actions.Do(ctx)
	if errs, ok := err.(docstore.ActionListError); ok {
		hasErrors := make(map[int]bool)
		for _, err := range errs {
			hasErrors[err.Index] = true
		}
		return hasErrors, err
	}
	return nil, err
}

func (d *docstoreStorage) BatchTokenGet(ctx context.Context, tokens []*tokenEntity) (errs map[int]bool, err error) {
	actions := d.collection.Actions()
	for i := range tokens {
		actions = actions.Get(tokens[i])
	}
	err = actions.Do(ctx)
	if errs, ok := err.(docstore.ActionListError); ok {
		hasErrors := make(map[int]bool)
		for _, err := range errs {
			hasErrors[err.Index] = true
		}
		return hasErrors, err
	}
	return nil, err
}

func (d *docstoreStorage) DeleteDoc(doc *Document) error {
	return d.collection.Delete(d.ctx, doc)
}

func (d *docstoreStorage) DeleteDocKey(docKey *documentKey) error {
	return d.collection.Delete(d.ctx, docKey)
}

func (d *docstoreStorage) DeleteTag(tag *tagEntity) error {
	return d.collection.Delete(d.ctx, tag)
}

func (d *docstoreStorage) DeleteToken(token *tokenEntity) error {
	return d.collection.Delete(d.ctx, token)
}

func (d *docstoreStorage) Close() error {
	return d.collection.Close()
}

func newDocstoreStorage(ctx context.Context, collection *docstore.Collection, index string, concurrency int) (Storage, error) {
	result := &docstoreStorage{
		ctx:        ctx,
		collection: collection,
	}
	result.counter = cloudcounter.NewCounter(collection, cloudcounter.Option{
		Concurrency: concurrency,
		Prefix:      index + "c",
	})
	err := result.counter.Register(ctx, documentID)
	if err != nil {
		return nil, err
	}
	err = result.counter.Register(ctx, documentCount)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type localStorage struct {
	DefaultLanguage string
	LastDocIndex    int
	DocumentCount   int
	Docs            map[string]interface{}
	lock            *sync.RWMutex
}

func (l *localStorage) DocCount() (int, error) {
	return l.DocumentCount, nil
}

func (l *localStorage) DecrementDocCount() error {
	l.DocumentCount--
	return nil
}

func (l *localStorage) Context() context.Context {
	return context.Background()
}

var ErrBatchGet = errors.New("batch get error")

func (l *localStorage) BatchDocGet(ctx context.Context, docs []*Document) (map[int]bool, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	errors := make(map[int]bool)
	for i, doc := range docs {
		err := l.GetDoc(doc)
		if err != nil {
			errors[i] = true
		}
	}
	if len(errors) == 0 {
		return nil, nil
	}
	return errors, ErrBatchGet
}

func (l *localStorage) BatchTagGet(ctx context.Context, tags []*tagEntity) (map[int]bool, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	errors := make(map[int]bool)
	for i, tag := range tags {
		err := l.GetTag(tag)
		if err != nil {
			errors[i] = true
		}
	}
	if len(errors) == 0 {
		return nil, nil
	}
	return errors, ErrBatchGet
}

func (l *localStorage) BatchTokenGet(ctx context.Context, tokens []*tokenEntity) (map[int]bool, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	errors := make(map[int]bool)
	for i, token := range tokens {
		err := l.GetToken(token)
		if err != nil {
			errors[i] = true
		}
	}
	if len(errors) == 0 {
		return nil, nil
	}
	return errors, ErrBatchGet
}

func newLocalIndex() *localStorage {
	return &localStorage{
		Docs: make(map[string]interface{}),
		lock: &sync.RWMutex{},
	}
}

func (l *localStorage) IncrementDocID() (int, error) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.LastDocIndex++
	return l.LastDocIndex, nil
}

func (l *localStorage) IncrementDocCount() (int, error) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.DocumentCount++
	return l.DocumentCount, nil
}

func (l *localStorage) Create(id string, doc interface{}) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.Docs[id] = doc
	return nil
}

func (l *localStorage) Replace(id string, doc interface{}) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	_, ok := l.Docs[id]
	if ok {
		l.Docs[id] = doc
		return nil
	}
	return errors.New("not found")
}

func (l *localStorage) GetDoc(doc *Document) error {
	l.lock.RLock()
	defer l.lock.RUnlock()
	res, ok := l.Docs[doc.ID]
	if ok {
		*doc = *(res.(*Document))
		return nil
	}
	return errors.New("not found")
}

func (l *localStorage) GetDocKey(docKey *documentKey) error {
	l.lock.RLock()
	defer l.lock.RUnlock()
	res, ok := l.Docs[docKey.ID]
	if ok {
		*docKey = *(res.(*documentKey))
		return nil
	}
	return errors.New("not found")
}

func (l *localStorage) GetTag(tag *tagEntity) error {
	l.lock.RLock()
	defer l.lock.RUnlock()
	res, ok := l.Docs[tag.ID]
	if ok {
		*tag = *(res.(*tagEntity))
		return nil
	}
	return errors.New("not found")
}

func (l *localStorage) GetToken(token *tokenEntity) error {
	l.lock.RLock()
	defer l.lock.RUnlock()
	res, ok := l.Docs[token.ID]
	if ok {
		*token = *(res.(*tokenEntity))
		return nil
	}
	return errors.New("not found")
}

func (l *localStorage) DeleteDoc(doc *Document) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	delete(l.Docs, doc.ID)
	return nil
}

func (l *localStorage) DeleteDocKey(docKey *documentKey) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	delete(l.Docs, docKey.ID)
	return nil
}

func (l *localStorage) DeleteTag(tag *tagEntity) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	delete(l.Docs, tag.ID)
	return nil
}

func (l *localStorage) DeleteToken(token *tokenEntity) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	delete(l.Docs, token.ID)
	return nil
}

func (l *localStorage) Close() error {
	l.lock.Lock()
	defer l.lock.Unlock()
	return nil
}

// WriteIndex writes content into io.Writer
func (l *localStorage) WriteIndex(w io.Writer) error {
	l.lock.RLock()
	defer l.lock.RUnlock()
	cp := &localStorage{
		DefaultLanguage: l.DefaultLanguage,
		LastDocIndex:    l.LastDocIndex,
		DocumentCount:   l.DocumentCount,
		Docs:            l.Docs,
	}
	enc := gob.NewEncoder(w)
	return enc.Encode(cp)
}

// loadIndex loads index from io.Reader
func (l *localStorage) ReadIndex(r io.Reader) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	cp := &localStorage{}
	dec := gob.NewDecoder(r)
	err := dec.Decode(cp)
	if err != nil {
		return err
	}
	l.DefaultLanguage = cp.DefaultLanguage
	l.DocumentCount = cp.DocumentCount
	l.LastDocIndex = cp.LastDocIndex
	l.Docs = cp.Docs
	return nil
}

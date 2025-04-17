package core

import (
	"sort"
	"sync"
	"time"

	"github.com/probe-lab/akai/db/models"
)

// itemSet is a simple queue of Items that allows rapid access to content through maps,
// while being able to sort the array by closer next ping time to determine which is
// the next item to be sampled
// adaptation of https://github.com/cortze/ipfs-cid-hoarder/blob/master/pkg/hoarder/cid_set.go
type dasItemSet struct {
	sync.RWMutex

	itemMap   map[string]*models.SamplingItem
	itemArray sortableItemArray

	pointer int
}

type sortableItemArray []*models.SamplingItem

// newItemSet creates a new itemSet
func newDASItemSet() *dasItemSet {
	return &dasItemSet{
		itemMap:   make(map[string]*models.SamplingItem),
		itemArray: make([]*models.SamplingItem, 0),
		pointer:   -1,
	}
}

func (s *dasItemSet) isInit() bool {
	s.RLock()
	defer s.RUnlock()

	return len(s.itemArray) > 0
}

// addItem adds the given item to the item set. If the item already
// exists in the set it won't be added nor overwritten. Instead, the return value
// will be false. You can use this signal to remove the item with [removeItem]
// and then add it again. The method returns true if the given item was added
// to the set.
func (s *dasItemSet) addItem(c *models.SamplingItem) bool {
	s.Lock()
	defer s.Unlock()

	// don't add the item if it already exists
	if _, ok := s.itemMap[c.Key]; ok {
		return false
	}

	s.itemMap[c.Key] = c
	s.itemArray = append(s.itemArray, c)

	return true
}

func (s *dasItemSet) removeItem(key string) bool {
	s.Lock()
	defer s.Unlock()

	if _, found := s.itemMap[key]; !found {
		return false
	}

	delete(s.itemMap, key)

	for idx, c := range s.itemArray {
		if c.Key == key {
			s.itemArray = append(s.itemArray[:idx], s.itemArray[(idx+1):]...)
			return true
		}
	}

	return false
}

// Iterators

func (s *dasItemSet) Next() *models.SamplingItem {
	s.Lock()
	defer s.Unlock()

	if s.pointer >= len(s.itemArray) || s.pointer < 0 {
		return nil
	}

	item := s.itemArray[s.pointer]
	s.pointer++
	return item
}

// common usage

// NextVisitTime returns the time until the next visit + bool to see if we need
// to sort back the item set
func (s *dasItemSet) NextVisitTime() (time.Time, bool) {
	s.RLock()
	defer s.RUnlock()

	// we only have a single sample
	if s.pointer == 0 && len(s.itemArray) == 1 {
		return s.itemArray[s.pointer].NextVisit, true
	}

	// we've reached the limit of the set
	if s.pointer+1 >= len(s.itemArray) {
		return time.Time{}, true
	}

	// return the next item's visit time
	return s.itemArray[s.pointer+1].NextVisit, false
}

func (s *dasItemSet) getItem(cStr string) (*models.SamplingItem, bool) {
	s.RLock()
	defer s.RUnlock()

	c, ok := s.itemMap[cStr]
	return c, ok
}

func (s *dasItemSet) getItemList() []*models.SamplingItem {
	s.RLock()
	defer s.RUnlock()

	cidList := make([]*models.SamplingItem, 0, len(s.itemArray))
	cidList = append(cidList, s.itemArray...)
	return cidList
}

func (s *dasItemSet) SortItemList() {
	s.Lock()
	defer s.Unlock()

	var ssa sortableItemArray = s.itemArray
	sort.Sort(ssa)
	s.pointer = 0
}

// Swap is part of sort.Interface.
func (s sortableItemArray) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less is part of sort.Interface. We use c.PeerList.NextConnection as the value to sort by.
func (s sortableItemArray) Less(i, j int) bool {
	return s[i].NextVisit.Before(s[j].NextVisit)
}

// Len is part of sort.Interface. We use the peer list to get the length of the array.
func (s sortableItemArray) Len() int {
	return len(s)
}

func (s *dasItemSet) Len() int {
	s.RLock()
	defer s.RUnlock()

	return len(s.itemArray)
}

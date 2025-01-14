package core

import (
	"sort"
	"sync"
	"time"

	"github.com/probe-lab/akai/db/models"
)

// segmentSet is a simple queue of Segments that allows rapid access to content through maps,
// while being able to sort the array by closer next ping time to determine which is
// the next segment to be sampled
// adaptation of https://github.com/cortze/ipfs-cid-hoarder/blob/master/pkg/hoarder/cid_set.go
type segmentSet struct {
	sync.RWMutex

	segmentMap   map[string]*models.AgnosticSegment
	segmentArray sortableSegmentArray

	pointer int
}

type sortableSegmentArray []*models.AgnosticSegment

// newSegmentSet creates a new segmentSet
func newSegmentSet() *segmentSet {
	return &segmentSet{
		segmentMap:   make(map[string]*models.AgnosticSegment),
		segmentArray: make([]*models.AgnosticSegment, 0),
		pointer:      -1,
	}
}

func (s *segmentSet) isInit() bool {
	s.RLock()
	defer s.RUnlock()

	return len(s.segmentArray) > 0
}

func (s *segmentSet) isSegmentAlready(c string) bool {
	s.RLock()
	defer s.RUnlock()

	_, ok := s.segmentMap[c]
	return ok
}

// addSegment adds the given segment to the segment set. If the segment already
// exists in the set it won't be added nor overwritten. Instead, the return value
// will be false. You can use this signal to remove the segment with [removeSegment]
// and then add it again. The method returns true if the given segment was added
// to the set.
func (s *segmentSet) addSegment(c *models.AgnosticSegment) bool {
	s.Lock()
	defer s.Unlock()

	// don't add the segment if it already exists
	if _, ok := s.segmentMap[c.Key]; ok {
		return false
	}

	s.segmentMap[c.Key] = c
	s.segmentArray = append(s.segmentArray, c)

	return true
}

func (s *segmentSet) removeSegment(key string) bool {
	s.Lock()
	defer s.Unlock()

	if _, found := s.segmentMap[key]; !found {
		return false
	}

	delete(s.segmentMap, key)

	for idx, c := range s.segmentArray {
		if c.Key == key {
			s.segmentArray = append(s.segmentArray[:idx], s.segmentArray[(idx+1):]...)
			return true
		}
	}

	return false
}

// Iterators

func (s *segmentSet) Next() *models.AgnosticSegment {
	s.Lock()
	defer s.Unlock()

	if s.pointer >= len(s.segmentArray) || s.pointer < 0 {
		return nil
	}

	segment := s.segmentArray[s.pointer]
	s.pointer++
	return segment
}

// common usage

// NextVisitTime returns the time until the next visit + bool to see if we need
// to sort back the segment set
func (s *segmentSet) NextVisitTime() (time.Time, bool) {
	s.RLock()
	defer s.RUnlock()

	// we only have a single sample
	if s.pointer == 0 && len(s.segmentArray) == 1 {
		return s.segmentArray[s.pointer].NextVisit, true
	}

	// we've reached the limit of the set
	if s.pointer+1 >= len(s.segmentArray) {
		return time.Time{}, true
	}

	// return the next segment's visit time
	return s.segmentArray[s.pointer+1].NextVisit, false
}

func (s *segmentSet) getSegment(cStr string) (*models.AgnosticSegment, bool) {
	s.RLock()
	defer s.RUnlock()

	c, ok := s.segmentMap[cStr]
	return c, ok
}

func (s *segmentSet) getSegmentList() []*models.AgnosticSegment {
	s.RLock()
	defer s.RUnlock()

	cidList := make([]*models.AgnosticSegment, 0, len(s.segmentArray))
	cidList = append(cidList, s.segmentArray...)
	return cidList
}

func (s *segmentSet) SortSegmentList() {
	s.Lock()
	defer s.Unlock()

	var ssa sortableSegmentArray = s.segmentArray
	sort.Sort(ssa)
	s.pointer = 0
}

// Swap is part of sort.Interface.
func (s sortableSegmentArray) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less is part of sort.Interface. We use c.PeerList.NextConnection as the value to sort by.
func (s sortableSegmentArray) Less(i, j int) bool {
	return s[i].NextVisit.Before(s[j].NextVisit)
}

// Len is part of sort.Interface. We use the peer list to get the length of the array.
func (s sortableSegmentArray) Len() int {
	return len(s)
}

func (s *segmentSet) Len() int {
	s.RLock()
	defer s.RUnlock()

	return len(s.segmentArray)
}

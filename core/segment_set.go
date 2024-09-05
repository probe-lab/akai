package core

import (
	"sort"
	"sync"
	"time"

	"github.com/probe-lab/akai/db/models"
)

// segmentSet is a simple queue of Segments that allows rapid access to content through maps,
// while being abel to sort the array by closer next ping time to determine which is
// the next segment to be sampled
// adaptation of https://github.com/cortze/ipfs-cid-hoarder/blob/master/pkg/hoarder/cid_set.go
type segmentSet struct {
	sync.RWMutex

	segmentMap   map[string]*models.AgnosticSegment
	segmentArray []*models.AgnosticSegment

	init    bool
	pointer int
}

// newSegmentSet creates a new segmentSet
func newSegmentSet() *segmentSet {
	return &segmentSet{
		segmentMap:   make(map[string]*models.AgnosticSegment),
		segmentArray: make([]*models.AgnosticSegment, 0),
		init:         false,
		pointer:      -1,
	}
}

func (s *segmentSet) isInit() bool {
	return s.init
}

func (s *segmentSet) isSegmentAlready(c string) bool {
	s.RLock()
	defer s.RUnlock()

	_, ok := s.segmentMap[c]
	return ok
}

func (s *segmentSet) addSegment(c *models.AgnosticSegment) {
	s.Lock()
	defer s.Unlock()

	s.segmentMap[c.Key] = c
	s.segmentArray = append(s.segmentArray, c)

	if !s.init {
		s.init = true
	}
}

func (s *segmentSet) removeSegment(cStr string) {
	delete(s.segmentMap, cStr)
	// check if len of the s.eue is only one
	if s.Len() == 1 {
		s.segmentArray = make([]*models.AgnosticSegment, 0)
		return
	}
	item := -1
	for idx, c := range s.segmentArray {
		if c.Key == cStr {
			item = idx
			break
		}
	}
	// check if the item was found
	if item >= 0 {
		s.segmentArray = append(s.segmentArray[:item], s.segmentArray[(item+1):]...)
	}
}

// Iterators

func (s *segmentSet) Next() bool {
	s.RLock()
	defer s.RUnlock()
	return s.pointer < s.Len() && s.pointer >= 0
}

func (s *segmentSet) Segment() *models.AgnosticSegment {
	if s.pointer >= s.Len() {
		return nil
	}
	s.Lock()
	defer func() {
		s.pointer++
		s.Unlock()
	}()
	return s.segmentArray[s.pointer]
}

// common usage

// time to next visit + bool to see if we need to sort back the segment set
func (s *segmentSet) NextVisitTime() (time.Time, bool) {
	s.RLock()
	defer s.RUnlock()

	// we only have on single sample
	if s.pointer == 0 && s.Len() == 1 {
		return s.segmentArray[s.pointer].NextVisit, true
	}

	// we've reached the limit of the set
	if s.pointer >= s.Len() {
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
	cidList := make([]*models.AgnosticSegment, s.Len())
	cidList = append(cidList, s.segmentArray...)
	return cidList
}

func (s *segmentSet) SortSegmentList() {
	sort.Sort(s)
	s.pointer = 0
}

// Swap is part of sort.Interface.
func (s *segmentSet) Swap(i, j int) {
	s.Lock()
	defer s.Unlock()

	s.segmentArray[i], s.segmentArray[j] = s.segmentArray[j], s.segmentArray[i]
}

// Less is part of sort.Interface. We use c.PeerList.NextConnection as the value to sort by.
func (s *segmentSet) Less(i, j int) bool {
	s.RLock()
	defer s.RUnlock()

	return s.segmentArray[i].NextVisit.Before(s.segmentArray[j].NextVisit)
}

// Len is part of sort.Interface. We use the peer list to get the length of the array.
func (s *segmentSet) Len() int {
	s.RLock()
	defer s.RUnlock()

	return len(s.segmentArray)
}

func (s *segmentSet) NextValidTimeToPing() (time.Time, bool) {
	var validTime time.Time
	s.RLock()
	defer s.RUnlock()
	for _, cid := range s.segmentArray {
		nextping := cid.NextVisit
		// asume that the array is sorted
		if validTime.IsZero() && !nextping.IsZero() {
			validTime = nextping
			return validTime, true
		}
	}
	return validTime, false
}

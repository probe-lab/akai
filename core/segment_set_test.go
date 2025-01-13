package core

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/probe-lab/akai/db/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SegmentSet(t *testing.T) {
	segmentSet := newSegmentSet()

	segment1 := models.AgnosticSegment{
		Timestamp:   time.Now(),
		Key:         "0xKEY",
		BlobNumber:  1,
		Row:         1,
		Column:      1,
		SampleUntil: time.Now().Add(1 * time.Minute),
		NextVisit:   time.Now().Add(35 * time.Second),
	}

	segment2 := models.AgnosticSegment{
		Timestamp:   time.Now(),
		Key:         "0xKEY2",
		BlobNumber:  1,
		Row:         2,
		Column:      1,
		SampleUntil: time.Now().Add(1 * time.Minute),
		NextVisit:   time.Now().Add(25 * time.Second),
	}

	// TODO: currently here to use the methods
	seg, exists := segmentSet.getSegment(segment1.Key)
	require.Equal(t, (*models.AgnosticSegment)(nil), seg)
	require.Equal(t, false, exists)

	segs := segmentSet.getSegmentList()
	require.Equal(t, 0, len(segs))

	segmentSet.addSegment(&segment1)
	segmentSet.addSegment(&segment2)
	segmentSet.SortSegmentList()

	// get first sorted segment
	nextSegment := segmentSet.Next()
	require.Equal(t, segment2.Key, nextSegment.Key)

	// get second sorted segment
	nextSegment = segmentSet.Next()
	require.Equal(t, segment1.Key, nextSegment.Key)
}

func Test_newSegmentSet(t *testing.T) {
	ss := newSegmentSet()
	assert.NotNil(t, ss.segmentMap)
	assert.NotNil(t, ss.segmentArray)
	assert.Equal(t, -1, ss.pointer)

	assert.False(t, ss.isInit())
}

func Test_addSegment_removeSegment_getSegment(t *testing.T) {
	ss := newSegmentSet()

	testSeg := &models.AgnosticSegment{
		Key: "some-key",
	}

	added := ss.addSegment(testSeg)
	assert.Equal(t, 1, len(ss.segmentMap))
	assert.Equal(t, 1, len(ss.segmentArray))
	assert.True(t, ss.isInit())
	assert.True(t, added)

	segment, found := ss.getSegment(testSeg.Key)
	assert.Equal(t, testSeg, segment)
	assert.True(t, found)

	added = ss.addSegment(testSeg)
	assert.Equal(t, 1, len(ss.segmentMap))
	assert.Equal(t, 1, len(ss.segmentArray))
	assert.True(t, ss.isInit())
	assert.False(t, added)

	segment, found = ss.getSegment(testSeg.Key)
	assert.Equal(t, testSeg, segment)
	assert.True(t, found)

	removed := ss.removeSegment(testSeg.Key)
	assert.Equal(t, 0, len(ss.segmentMap))
	assert.Equal(t, 0, len(ss.segmentArray))
	assert.False(t, ss.isInit())
	assert.True(t, removed)

	segment, found = ss.getSegment(testSeg.Key)
	assert.Nil(t, segment)
	assert.False(t, found)
}

func Test_sortSegments(t *testing.T) {
	ss := newSegmentSet()

	now := time.Now()
	segments := []*models.AgnosticSegment{
		{
			Key:       "segment_2",
			NextVisit: now.Add(2 * time.Minute),
		},
		{
			Key:       "segment_0",
			NextVisit: now.Add(0 * time.Minute),
		},
		{
			Key:       "segment_1",
			NextVisit: now.Add(1 * time.Minute),
		},
	}

	for _, seg := range segments {
		ss.addSegment(seg)
	}

	ss.SortSegmentList()

	for i, segment := range ss.getSegmentList() {
		switch i {
		case 0:
			assert.Equal(t, segments[1], segment)
		case 1:
			assert.Equal(t, segments[2], segment)
		case 2:
			assert.Equal(t, segments[0], segment)
		}
	}
}

func Test_async_segment_set(t *testing.T) {
	ss := newSegmentSet()

	segmentCount := 1_000

	now := time.Now()
	segmentsToAdd := make([]*models.AgnosticSegment, segmentCount)
	segmentsToRemove := make([]*models.AgnosticSegment, segmentCount)
	for i := 0; i < segmentCount; i++ {
		segment := &models.AgnosticSegment{
			Key:       "segment_" + strconv.Itoa(i),
			NextVisit: now.Add(time.Duration(i) * time.Minute),
		}
		segmentsToAdd[i] = segment
		segmentsToRemove[i] = segment
	}

	rand.Shuffle(len(segmentsToAdd), func(i, j int) {
		segmentsToAdd[i], segmentsToAdd[j] = segmentsToAdd[j], segmentsToAdd[i]
	})

	rand.Shuffle(len(segmentsToRemove), func(i, j int) {
		segmentsToRemove[i], segmentsToRemove[j] = segmentsToRemove[j], segmentsToRemove[i]
	})

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		for _, seg := range segmentsToAdd {
			ss.addSegment(seg)
		}
	}()

	go func() {
		defer wg.Done()
		for _, seg := range segmentsToRemove {
			ss.removeSegment(seg.Key)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < segmentCount; i++ {
			ss.SortSegmentList()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < segmentCount; i++ {
			ss.Next()
		}
	}()

	wg.Wait()
}

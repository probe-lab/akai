package core

import (
	"testing"
	"time"

	"github.com/probe-lab/akai/db/models"
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
	isThereNext := segmentSet.Next()
	require.Equal(t, true, isThereNext)
	nextSegment := segmentSet.Segment()
	require.Equal(t, segment2.Key, nextSegment.Key)

	// get second sorted segment
	isThereNext = segmentSet.Next()
	require.Equal(t, true, isThereNext)
	nextSegment = segmentSet.Segment()
	require.Equal(t, segment1.Key, nextSegment.Key)
}

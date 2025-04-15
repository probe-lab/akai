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

func Test_ItemSet(t *testing.T) {
	ItemSet := newDASItemSet()

	Item1 := models.SamplingItem{
		Timestamp:   time.Now(),
		Key:         "0xKEY",
		BlockLink:   1,
		DASRow:      1,
		DASColumn:   1,
		SampleUntil: time.Now().Add(1 * time.Minute),
		NextVisit:   time.Now().Add(35 * time.Second),
	}

	Item2 := models.SamplingItem{
		Timestamp:   time.Now(),
		Key:         "0xKEY2",
		BlockLink:   1,
		DASRow:      2,
		DASColumn:   1,
		SampleUntil: time.Now().Add(1 * time.Minute),
		NextVisit:   time.Now().Add(25 * time.Second),
	}

	// TODO: currently here to use the methods
	seg, exists := ItemSet.getItem(Item1.Key)
	require.Equal(t, (*models.SamplingItem)(nil), seg)
	require.Equal(t, false, exists)

	segs := ItemSet.getItemList()
	require.Equal(t, 0, len(segs))

	ItemSet.addItem(&Item1)
	ItemSet.addItem(&Item2)
	ItemSet.SortItemList()

	// get first sorted Item
	nextItem := ItemSet.Next()
	require.Equal(t, Item2.Key, nextItem.Key)

	// get second sorted Item
	nextItem = ItemSet.Next()
	require.Equal(t, Item1.Key, nextItem.Key)
}

func Test_newItemSet(t *testing.T) {
	ss := newDASItemSet()
	assert.NotNil(t, ss.itemMap)
	assert.NotNil(t, ss.itemArray)
	assert.Equal(t, -1, ss.pointer)

	assert.False(t, ss.isInit())
}

func Test_addItem_removeItem_getItem(t *testing.T) {
	ss := newDASItemSet()

	testSeg := &models.SamplingItem{
		Key: "some-key",
	}

	added := ss.addItem(testSeg)
	assert.Equal(t, 1, len(ss.itemMap))
	assert.Equal(t, 1, len(ss.itemArray))
	assert.True(t, ss.isInit())
	assert.True(t, added)

	Item, found := ss.getItem(testSeg.Key)
	assert.Equal(t, testSeg, Item)
	assert.True(t, found)

	added = ss.addItem(testSeg)
	assert.Equal(t, 1, len(ss.itemMap))
	assert.Equal(t, 1, len(ss.itemArray))
	assert.True(t, ss.isInit())
	assert.False(t, added)

	Item, found = ss.getItem(testSeg.Key)
	assert.Equal(t, testSeg, Item)
	assert.True(t, found)

	removed := ss.removeItem(testSeg.Key)
	assert.Equal(t, 0, len(ss.itemMap))
	assert.Equal(t, 0, len(ss.itemArray))
	assert.False(t, ss.isInit())
	assert.True(t, removed)

	Item, found = ss.getItem(testSeg.Key)
	assert.Nil(t, Item)
	assert.False(t, found)
}

func Test_sortItems(t *testing.T) {
	ss := newDASItemSet()

	now := time.Now()
	Items := []*models.SamplingItem{
		{
			Key:       "Item_2",
			NextVisit: now.Add(2 * time.Minute),
		},
		{
			Key:       "Item_0",
			NextVisit: now.Add(0 * time.Minute),
		},
		{
			Key:       "Item_1",
			NextVisit: now.Add(1 * time.Minute),
		},
	}

	for _, seg := range Items {
		ss.addItem(seg)
	}

	ss.SortItemList()

	for i, Item := range ss.getItemList() {
		switch i {
		case 0:
			assert.Equal(t, Items[1], Item)
		case 1:
			assert.Equal(t, Items[2], Item)
		case 2:
			assert.Equal(t, Items[0], Item)
		}
	}
}

func Test_async_Item_set(t *testing.T) {
	ss := newDASItemSet()

	ItemCount := 1_000

	now := time.Now()
	ItemsToAdd := make([]*models.SamplingItem, ItemCount)
	ItemsToRemove := make([]*models.SamplingItem, ItemCount)
	for i := 0; i < ItemCount; i++ {
		Item := &models.SamplingItem{
			Key:       "Item_" + strconv.Itoa(i),
			NextVisit: now.Add(time.Duration(i) * time.Minute),
		}
		ItemsToAdd[i] = Item
		ItemsToRemove[i] = Item
	}

	rand.Shuffle(len(ItemsToAdd), func(i, j int) {
		ItemsToAdd[i], ItemsToAdd[j] = ItemsToAdd[j], ItemsToAdd[i]
	})

	rand.Shuffle(len(ItemsToRemove), func(i, j int) {
		ItemsToRemove[i], ItemsToRemove[j] = ItemsToRemove[j], ItemsToRemove[i]
	})

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		for _, seg := range ItemsToAdd {
			ss.addItem(seg)
		}
	}()

	go func() {
		defer wg.Done()
		for _, seg := range ItemsToRemove {
			ss.removeItem(seg.Key)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < ItemCount; i++ {
			ss.SortItemList()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < ItemCount; i++ {
			ss.Next()
		}
	}()

	wg.Wait()
}

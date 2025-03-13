package avail

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/probe-lab/akai/avail/api"
	"github.com/probe-lab/akai/config"
	log "github.com/sirupsen/logrus"
)

var availAPIretries = 5

type BlockRequester struct {
	httpAPICli *api.HTTPClient

	networkConfig        *config.NetworkConfiguration
	totalRequestedBlocks uint64
	lastBlockTracked     uint64

	trackDuration time.Duration
	trackInterval time.Duration

	consumers []BlockConsumer
}

func NewBlockRequester(
	apiCli *api.HTTPClient,
	networkConfig *config.NetworkConfiguration,
	consumers []BlockConsumer,
	trackDuration, trackInterval time.Duration) (*BlockRequester, error) {
	return &BlockRequester{
		httpAPICli:       apiCli,
		networkConfig:    networkConfig,
		lastBlockTracked: uint64(0),
		consumers:        consumers,
		trackDuration:    trackDuration,
		trackInterval:    trackInterval,
	}, nil
}

func (r *BlockRequester) Serve(ctx context.Context) error {
	go r.periodicBlockRequester(ctx)
	<-ctx.Done()
	log.Info("closing avail-block-requester")
	return nil
}

func (r *BlockRequester) AvailAPIhealthcheck(ctx context.Context) error {
	log.Info("avail API healthcheck...")
	for i := 0; i < availAPIretries; i++ {
		_, err := r.httpAPICli.GetV2Status(ctx)
		if err != nil {
			log.Errorf("healthchcheck %d failed %s", i, err.Error())
		} else {
			log.Info("got successfull connection to avail-light-api")
			return nil
		}
		time.Sleep(30 * time.Second)
	}
	return fmt.Errorf("no connection available to avail-light-api")
}

func (r *BlockRequester) periodicBlockRequester(ctx context.Context) {
	// request the genesis info
	availStatus, err := r.httpAPICli.GetV2Status(ctx)
	if err != nil {
		log.Error("requesting status to start block-requester ", err)
		return
	}
	// hot-fix: ensure that our last-seen is 1 block prior to the last available one
	// TODO: we will probably have to fetch from the DB which is the last tracked Block
	r.lastBlockTracked = availStatus.Blocks.AvailableRange.Last - 1 // request last available

	// calculate any waiting time in relation to the genesis (we want to be ~1 sec after the theoretical proposal)
	toNextProposal := secondsToNextProposal(r.networkConfig.GenesisTime)

	log.WithFields(log.Fields{
		"head_block":            availStatus.Blocks.Latest,
		"last_avail_block":      availStatus.Blocks.AvailableRange.Last,
		"time_to_next_proposal": toNextProposal,
	}).Info("launching block requester...")

	ticker := time.NewTicker(toNextProposal)
	select {
	case <-ticker.C:
		break
	case <-ctx.Done():
		log.Info("context died while waiting for first proposal")
		return
	}
	ticker.Stop()

	// logic to track or not to track blocks at specific intervals
	var trackSegmentsFlag bool = true
	trackDurationT := time.NewTicker(r.trackDuration)
	trackIntervalT := time.NewTicker(r.trackInterval)
	invertSegmentTrack := func() {
		*&trackSegmentsFlag = !trackSegmentsFlag
	}

	// once we are "on-time", keep requesting the block periodically
	extraDelay := 1 * time.Second
	proposalTicker := time.NewTicker(config.BlockIntervalTarget + extraDelay) // give 1 extra time to request next block
	lastReqT := time.Now()
	for {
		select {
		case <-trackDurationT.C:
			invertSegmentTrack()
			trackDurationT.Stop()

		case <-trackIntervalT.C:
			invertSegmentTrack()
			trackDurationT.Reset(r.trackDuration)

		case <-proposalTicker.C:
			headBlock, lastBlock, err := r.requestLastState(ctx)
			if err != nil {
				log.Error("requesting new block from block-tracker", err)
				return
			}
			log.WithFields(log.Fields{
				"last_block": lastBlock,
				"head_block": headBlock,
			}).Debug("avail-light client reports...")

			if !r.isBlockNew(lastBlock) {
				// should we wait 1 sec and try again?
				log.WithField("delay", extraDelay).Debug("no new-block found, requesting it again")
				proposalTicker.Reset(extraDelay)

			} else {
				for blockToRequest := r.lastBlockTracked + 1; blockToRequest <= lastBlock; blockToRequest++ {
					// update metrics
					r.totalRequestedBlocks++
					blockHeader, err := r.httpAPICli.GetV2BlockHeader(ctx, blockToRequest)
					if err != nil {
						log.WithField(
							"requested_block", blockToRequest,
						).Debug(errors.Wrap(err, "avail-light couldn't give block header"))
						continue
					}
					err = r.processNewBlock(ctx, blockHeader, lastReqT, trackSegmentsFlag)
					if err != nil {
						log.Error(err)
						return
					}
					lastReqT = time.Now()
				}
				proposalTicker.Reset(config.BlockIntervalTarget) // wait 20 seconds from last proposal
				// TODO: more spammy alternative
				// secondsToNextProposal(r.genesisTime) + extraDelay
			}

		case <-ctx.Done():
			log.Info("context died while waiting for next proposal")
			return
		}
	}
}

func (r *BlockRequester) processNewBlock(
	ctx context.Context,
	blockHeader api.V2BlockHeader,
	lastReqT time.Time,
	trackSegment bool) error {
	newBlockNot := &BlockNotification{
		RequestTime: time.Now(),
		BlockInfo:   blockHeader,
	}
	for _, consumer := range r.consumers {
		err := consumer.ProccessNewBlock(ctx, newBlockNot, lastReqT, trackSegment)
		if err != nil {
			return err
		}
	}
	// update our last seen to the last processed one
	r.lastBlockTracked = blockHeader.Number
	return nil
}

func (r *BlockRequester) requestLastState(ctx context.Context) (uint64, uint64, error) {
	// request the genesis info
	availStatus, err := r.httpAPICli.GetV2Status(ctx)
	if err != nil {
		return uint64(0), uint64(0), err
	}
	return availStatus.Blocks.Latest, availStatus.Blocks.AvailableRange.Last, nil
}

func (r *BlockRequester) isBlockNew(block uint64) bool {
	return block > r.lastBlockTracked
}

func timeSinceGenesis(genTime time.Time) time.Duration {
	return time.Since(genTime)
}

func secondsToNextProposal(genTime time.Time) time.Duration {
	sinceGen := timeSinceGenesis(genTime)
	timeToNextBlock := sinceGen % time.Duration(config.BlockIntervalTarget)
	return timeToNextBlock
}

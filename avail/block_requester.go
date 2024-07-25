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

type BlockRequester struct {
	httpApiCli *api.HttpClient

	genesisTime      time.Time
	lastBlockTracked uint64

	consumers []BlockConsumer
}

func NewBlockRequester(apiCli *api.HttpClient, network config.Network, consumers []BlockConsumer) (*BlockRequester, error) {
	genTime := api.GenesisTimeFromNetwork(network)
	if genTime.IsZero() {
		return nil, fmt.Errorf("empty genesis time (%s) for network %s", genTime, network.String())
	}

	return &BlockRequester{
		httpApiCli:       apiCli,
		genesisTime:      genTime,
		lastBlockTracked: uint64(0),
		consumers:        consumers,
	}, nil
}

func (r *BlockRequester) Serve(ctx context.Context) error {
	go r.periodicBlockRequester(ctx)
	<-ctx.Done()
	log.Info("closing avail-block-requester")
	return nil
}

func (r *BlockRequester) periodicBlockRequester(ctx context.Context) {
	// request the genesis info
	availStatus, err := r.httpApiCli.GetV2Status(ctx)
	if err != nil {
		log.Panic("requesting status to start block-requester", err)
	}
	// hot-fix: ensure that our last-seen is 1 block prior to the last available one
	// TODO: we will probably have to fetch from the DB which is the last tracked Block
	r.lastBlockTracked = availStatus.Blocks.AvailableRange.Last - 1 // request last available

	// calculate any waiting time in relation to the genesis (we want to be ~1 sec after the theoretical proposal)
	toNextProposal := secondsToNextProposal(r.genesisTime)

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

	// once we are "on-time", keep requesting the block periodically
	extraDelay := 1 * time.Second
	proposalTicker := time.NewTicker(api.BlockIntervalTarget + extraDelay) // give 1 extra time to request next block
	for {
		select {
		case <-proposalTicker.C:
			headBlock, lastBlock, err := r.requestLastState(ctx)
			if err != nil {
				log.Panic("requesting new block from block-tracker", err)
			}
			log.WithFields(log.Fields{
				"last_block": lastBlock,
				"head_block": headBlock,
			}).Debug("avail-light client reports...")

			if !r.isBlockNew(lastBlock) {
				// should we wait 1 sec and try again?
				log.WithField("delay", extraDelay).Warn("no new-block found, requesting it again")
				proposalTicker.Reset(extraDelay)

			} else {
				for blockToRequest := r.lastBlockTracked + 1; blockToRequest <= lastBlock; blockToRequest++ {
					blockHeader, err := r.httpApiCli.GetV2BlockHeader(ctx, blockToRequest)
					if err != nil {
						log.WithField(
							"requested_block", blockToRequest,
						).Error(errors.Wrap(err, "requesting new block"))
					}
					err = r.processNewBlock(ctx, blockHeader)
					if err != nil {
						log.Panic(err)

					}
				}
				proposalTicker.Reset(api.BlockIntervalTarget) // wait 20 seconds from last proposal
				// TODO: more spammy alternative
				// secondsToNextProposal(r.genesisTime) + extraDelay
			}

		case <-ctx.Done():
			log.Info("context died while waiting for next proposal")
			return
		}
	}
}

func (r *BlockRequester) processNewBlock(ctx context.Context, blockHeader api.V2BlockHeader) error {
	newBlockNot := &BlockNotification{
		RequestTime: time.Now(),
		BlockInfo:   blockHeader,
	}
	for _, consumer := range r.consumers {
		err := consumer.ProccessNewBlock(ctx, newBlockNot)
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
	availStatus, err := r.httpApiCli.GetV2Status(ctx)
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
	timeToNextBlock := sinceGen % time.Duration(api.BlockIntervalTarget)
	return timeToNextBlock
}

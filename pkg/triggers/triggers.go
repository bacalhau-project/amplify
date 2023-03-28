package triggers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bacalhau-project/amplify/pkg/util"
	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
)

var ErrURLNotSet = fmt.Errorf("IPFS-Search URL not set")
var ErrPeriodNotSet = fmt.Errorf("IPFS-Search period not set")

type IPFSSearchTrigger struct {
	URL    string
	Period time.Duration // Period to wait between requests to IPFS Search
}

func (t *IPFSSearchTrigger) Start(ctx context.Context, cidChan chan cid.Cid) error {
	if t.URL == "" {
		return ErrURLNotSet
	}
	if t.Period == 0*time.Second {
		return ErrPeriodNotSet
	}
	log.Ctx(ctx).Info().Str("url", t.URL).Str("period", t.Period.String()).Msg("Starting IPFS Search trigger")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(t.Period):
			r, err := http.Get(t.URL)
			if err != nil {
				log.Ctx(ctx).Warn().Err(err).Msg("error while fetching IPFS Search")
			}
			cids, err := parseIPFSSearchResponse(r)
			if err != nil {
				log.Ctx(ctx).Warn().Err(err).Msg("error while parsing IPFS Search response")
			}
			log.Ctx(ctx).Debug().Int("cids", len(cids)).Msg("Submitting IPFS Search results to Amplify")
			for _, c := range cids {
				cidChan <- c
			}
			log.Ctx(ctx).Debug().Msg("Sleeping for 1 minute before fetching IPFS Search again")
		}
	}
}

type ipfsSearchResult struct {
	Hash         string `json:"hash"`
	Title        string `json:"title"`
	CreationDate string `json:"creation_date"`
	Description  string `json:"description"`
	Type         string `json:"type"`
	Size         int    `json:"size"`
	FirstSeen    string `json:"first-seen"`
	LastSeen     string `json:"last-seen"`
	Score        int    `json:"score"`
	References   []struct {
		Name       string `json:"name"`
		ParentHash string `json:"parent_hash"`
	} `json:"references"`
	MimeType string `json:"mimetype"`
}

type ipfsSearchResponse struct {
	Total    int                `json:"total"`
	MaxScore int                `json:"max_score"`
	Hits     []ipfsSearchResult `json:"hits"`
}

func parseIPFSSearchResponse(r *http.Response) ([]cid.Cid, error) {
	defer r.Body.Close()
	var resp ipfsSearchResponse
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return nil, err
	}
	var cids []cid.Cid
	for _, r := range resp.Hits {
		var hit string
		if len(r.References) > 0 {
			hit = r.References[0].ParentHash
		} else {
			hit = r.Hash
		}
		c, err := cid.Decode(hit)
		if err != nil {
			log.Debug().Err(err).Msg("error while decoding CID")
			continue
		}
		cids = append(cids, c)
	}
	cids = util.Dedup(cids)
	return cids, nil
}

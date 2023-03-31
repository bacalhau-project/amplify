package triggers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/rs/zerolog/log"
	"gotest.tools/assert"
)

func TestIPFSSearchTrigger(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, testResult)
	}))
	defer ts.Close()
	trigger := IPFSSearchTrigger{
		URL:    ts.URL,
		Period: 1 * time.Millisecond,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	cidChan := make(chan cid.Cid)
	go func() {
		if err := trigger.Start(ctx, cidChan); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("error while starting IPFS Search trigger")
		}
	}()
	select {
	case <-ctx.Done():
		t.Fatal("context timed out")
	case c := <-cidChan:
		assert.Equal(t, "bafybeiaazowxsxznvnr2jnkiuvbesevoyrvae6zvgnikqot7czoiqkuhhm", c.String())
		cancel()
	}
}

func TestIPFSSearchTriggerError(t *testing.T) {
	trigger := IPFSSearchTrigger{}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := trigger.Start(ctx, nil)
	assert.Error(t, err, ErrURLNotSet.Error())
	trigger = IPFSSearchTrigger{URL: "hihi"}
	err = trigger.Start(ctx, nil)
	assert.Error(t, err, ErrPeriodNotSet.Error())
}

func TestIPFSSearchURLError(t *testing.T) {
	trigger := IPFSSearchTrigger{
		URL:    "invalid url",
		Period: 1 * time.Millisecond,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	cidChan := make(chan cid.Cid)
	// This should happily run. Before it was causing a panic due to the subsequent body.Close()
	go func() {
		if err := trigger.Start(ctx, cidChan); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("error while starting IPFS Search trigger")
		}
	}()
	<-ctx.Done()
}

const testResult = `{
	"total": 1,
	"max_score": 4,
	"hits": [
	  {
		"hash": "bafybeidnkkg4hnbbznzagx6fbp27p7tdh33jjhqmipbzbxbobduuczrihu",
		"title": "x6",
		"description": null,
		"type": "directory",
		"size": 133941956,
		"first-seen": "2023-03-24T14:41:38Z",
		"last-seen": "2023-03-24T14:41:38Z",
		"score": 4,
		"references": [
		  {
			"name": "x6",
			"parent_hash": "bafybeiaazowxsxznvnr2jnkiuvbesevoyrvae6zvgnikqot7czoiqkuhhm"
		  }
		],
		"mimetype": null
	  }
	],
	"page_size": 1,
	"page_count": 1
  }`

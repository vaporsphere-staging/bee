package epochs_test

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/feeds"
	"github.com/ethersphere/bee/pkg/feeds/epochs"
	"github.com/ethersphere/bee/pkg/storage"
	"github.com/ethersphere/bee/pkg/storage/mock"
)

func TestFinder(t *testing.T) {
	testf := func(t *testing.T, finderf func(storage.Getter, *feeds.Feed) feeds.Lookup) {
		t.Run("basic", func(t *testing.T) {
			testFinderBasic(t, finderf)
		})
		t.Run("fixed", func(t *testing.T) {
			testFinderFixIntervals(t, finderf)
		})
		t.Run("random", func(t *testing.T) {
			testFinderRandomIntervals(t, finderf)
		})
	}
	t.Run("sync", func(t *testing.T) {
		testf(t, epochs.NewFinder)
	})
	t.Run("async", func(t *testing.T) {
		testf(t, epochs.NewAsyncFinder)
	})
}

func testFinderBasic(t *testing.T, finderf func(storage.Getter, *feeds.Feed) feeds.Lookup) {
	storer := mock.NewStorer()
	topic := "testtopic"
	pk, _ := crypto.GenerateSecp256k1Key()
	signer := crypto.NewDefaultSigner(pk)

	updater, err := epochs.NewUpdater(storer, signer, topic)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	finder := finderf(storer, updater.Feed)
	t.Run("no update", func(t *testing.T) {
		ch, err := feeds.Latest(ctx, finder, 0)
		if err != nil {
			t.Fatal(err)
		}
		if ch != nil {
			t.Fatalf("expected no update, got addr %v", ch.Address())
		}
	})
	t.Run("first update root epoch", func(t *testing.T) {
		payload := []byte("payload")
		at := time.Now().Unix()
		err = updater.Update(ctx, at, payload)
		if err != nil {
			t.Fatal(err)
		}
		ch, err := feeds.Latest(ctx, finder, 0)
		if err != nil {
			t.Fatal(err)
		}
		if ch == nil {
			t.Fatalf("expected to find update, got none")
		}
		exp := payload
		ts, payload, err := feeds.FromChunk(ch)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(payload, exp) {
			t.Fatalf("result mismatch. want %8x... got %8x...", exp, payload)
		}
		if ts != uint64(at) {
			t.Fatalf("timestamp mismatch: expected %v, got %v", at, ts)
		}
	})
}

func testFinderFixIntervals(t *testing.T, finderf func(storage.Getter, *feeds.Feed) feeds.Lookup) {
	for _, tc := range []struct {
		count  int64
		step   int64
		offset int64
	}{
		{50, 1, 0},
		{50, 1, 10000},
		// {50, 100, 0},
		// {50, 100, 100000},
	} {
		t.Run(fmt.Sprintf("count=%d,step=%d,offset=%d", tc.count, tc.step, tc.offset), func(t *testing.T) {
			storer := mock.NewStorer()
			topic := "testtopic"
			pk, _ := crypto.GenerateSecp256k1Key()
			signer := crypto.NewDefaultSigner(pk)

			updater, err := epochs.NewUpdater(storer, signer, topic)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()

			payload := []byte("payload")
			for at := tc.offset; at < tc.offset+tc.count*tc.step; at += tc.step {
				err = updater.Update(ctx, at, payload)
				if err != nil {
					t.Fatal(err)
				}
			}
			finder := finderf(storer, updater.Feed)
			for at := tc.offset; at < tc.offset+tc.count*tc.step; at += tc.step {
				for after := tc.offset; after < at; after += tc.step {
					step := int64(1)
					if tc.step > 1 {
						step = tc.step / 4
					}
					for now := at; now < at+tc.step; now += step {
						ch, err := finder.At(ctx, now, after)
						if err != nil {
							t.Fatal(err)
						}
						if ch == nil {
							t.Fatalf("expected to find update, got none")
						}
						exp := payload
						ts, payload, err := feeds.FromChunk(ch)
						if err != nil {
							t.Fatal(err)
						}
						if !bytes.Equal(payload, exp) {
							t.Fatalf("payload mismatch: expected %x, got %x", exp, payload)
						}
						if ts != uint64(at) {
							t.Fatalf("timestamp mismatch: expected %v, got %v", at, ts)
						}
					}
				}
			}
		})
	}
}

func testFinderRandomIntervals(t *testing.T, finderf func(storage.Getter, *feeds.Feed) feeds.Lookup) {
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("random intervals %d", i), func(t *testing.T) {
			storer := mock.NewStorer()
			topic := "testtopic"
			pk, _ := crypto.GenerateSecp256k1Key()
			signer := crypto.NewDefaultSigner(pk)

			updater, err := epochs.NewUpdater(storer, signer, topic)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()

			payload := []byte("payload")
			var at int64
			ats := make([]int64, 100)
			for j := 0; j < 50; j++ {
				ats[j] = at
				at += int64(rand.Intn(1<<10) + 1)
				err = updater.Update(ctx, ats[j], payload)
				if err != nil {
					t.Fatal(err)
				}
			}
			finder := finderf(storer, updater.Feed)
			for j := 1; j < 49; j++ {
				diff := ats[j+1] - ats[j]
				for at := ats[j]; at < ats[j+1]; at += int64(rand.Intn(int(diff)) + 1) {
					for after := int64(0); after < at; after += int64(rand.Intn(int(at))) {
						ch, err := finder.At(ctx, at, after)
						if err != nil {
							t.Fatal(err)
						}
						if ch == nil {
							t.Fatalf("expected to find update, got none")
						}
						exp := payload
						ts, payload, err := feeds.FromChunk(ch)
						if err != nil {
							t.Fatal(err)
						}
						if !bytes.Equal(payload, exp) {
							t.Fatalf("payload mismatch: expected %x, got %x", exp, payload)
						}
						if ts != uint64(ats[j]) {
							t.Fatalf("timestamp mismatch: expected %v, got %v", ats[j], ts)
						}
					}
				}
			}
		})
	}
}
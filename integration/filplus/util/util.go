package util

import (
	"context"
	"github.com/data-preservation-programs/RetrievalBot/pkg/convert"
	"github.com/data-preservation-programs/RetrievalBot/pkg/env"
	"github.com/data-preservation-programs/RetrievalBot/pkg/model"
	"github.com/data-preservation-programs/RetrievalBot/pkg/requesterror"
	"github.com/data-preservation-programs/RetrievalBot/pkg/resolver"
	"github.com/data-preservation-programs/RetrievalBot/pkg/task"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
	"strconv"
	"time"
)

var logger = logging.Logger("addTasks")

//nolint:nonamedreturns
func AddTasks(ctx context.Context,
	requester string,
	ipInfo resolver.IPInfo,
	documents []model.DealState,
	locationResolver resolver.LocationResolver,
	providerResolver resolver.ProviderResolver) (tasks []interface{}, results []interface{}) {
	// Insert the documents into task queue
	for _, document := range documents {
		// If the label is a correct CID, assume it is the payload CID and try GraphSync and Bitswap retrieval
		labelCID, err := cid.Decode(document.Label)
		if err != nil {
			logger.With("label", document.Label, "deal_id", document.DealID).
				Debug("failed to decode label as CID")
			continue
		}

		isPayloadCID := true
		// Skip graphsync and bitswap if the cid is not decodable, i.e. it is a pieceCID
		if !slices.Contains([]uint64{cid.Raw, cid.DagCBOR, cid.DagProtobuf, cid.DagJSON, cid.DagJOSE},
			labelCID.Prefix().Codec) {
			logger.With("provider", document.Provider, "deal_id", document.DealID,
				"label", document.Label, "codec", labelCID.Prefix().Codec).
				Info("Skip Bitswap and Graphsync because the Label is likely not a payload CID")
			isPayloadCID = false
		}

		providerInfo, err := providerResolver.ResolveProvider(ctx, document.Provider)
		if err != nil {
			logger.With("provider", document.Provider, "deal_id", document.DealID).
				Error("failed to resolve provider")
			continue
		}

		location, err := locationResolver.ResolveMultiaddrsBytes(ctx, providerInfo.Multiaddrs)
		if err != nil {
			if errors.As(err, &requesterror.BogonIPError{}) ||
				errors.As(err, &requesterror.InvalidIPError{}) ||
				errors.As(err, &requesterror.HostLookupError{}) ||
				errors.As(err, &requesterror.NoValidMultiAddrError{}) {
				results = addErrorResults(requester, ipInfo, results, document, providerInfo, location,
					task.NoValidMultiAddrs, err.Error())
			} else {
				logger.With("provider", document.Provider, "deal_id", document.DealID, "err", err).
					Error("failed to resolve provider location")
			}
			continue
		}

		_, err = peer.Decode(providerInfo.PeerId)
		if err != nil {
			logger.With("provider", document.Provider, "deal_id", document.DealID, "peerID", providerInfo.PeerId,
				"err", err).
				Info("failed to decode peerID")
			results = addErrorResults(requester, ipInfo, results, document, providerInfo, location,
				task.InvalidPeerID, err.Error())
			continue
		}

		if isPayloadCID {
			for _, module := range []task.ModuleName{task.GraphSync, task.Bitswap} {
				tasks = append(tasks, task.Task{
					Requester: requester,
					Module:    module,
					Metadata: map[string]string{
						"deal_id":       strconv.Itoa(int(document.DealID)),
						"client":        document.Client,
						"assume_label":  "true",
						"retrieve_type": "root_block"},
					Provider: task.Provider{
						ID:         document.Provider,
						PeerID:     providerInfo.PeerId,
						Multiaddrs: convert.MultiaddrsBytesToStringArraySkippingError(providerInfo.Multiaddrs),
						City:       location.City,
						Region:     location.Region,
						Country:    location.Country,
						Continent:  location.Continent,
					},
					Content: task.Content{
						CID: document.Label,
					},
					CreatedAt: time.Now().UTC(),
					Timeout:   env.GetDuration(env.FilplusIntegrationTaskTimeout, 15*time.Second),
				})
			}
		}

		tasks = append(tasks, task.Task{
			Requester: requester,
			Module:    task.HTTP,
			Metadata: map[string]string{
				"deal_id":       strconv.Itoa(int(document.DealID)),
				"client":        document.Client,
				"retrieve_type": "piece",
				"retrieve_size": "1048576"},
			Provider: task.Provider{
				ID:         document.Provider,
				PeerID:     providerInfo.PeerId,
				Multiaddrs: convert.MultiaddrsBytesToStringArraySkippingError(providerInfo.Multiaddrs),
				City:       location.City,
				Region:     location.Region,
				Country:    location.Country,
				Continent:  location.Continent,
			},
			Content: task.Content{
				CID: document.PieceCID,
			},
			CreatedAt: time.Now().UTC(),
			Timeout:   env.GetDuration(env.FilplusIntegrationTaskTimeout, 15*time.Second),
		})
	}
	logger.With("count", len(tasks)).Info("inserted tasks")
	//nolint:nakedret
	return
}

var moduleMetadataMap = map[task.ModuleName]map[string]string{
	task.GraphSync: {
		"assume_label":  "true",
		"retrieve_type": "root_block",
	},
	task.Bitswap: {
		"assume_label":  "true",
		"retrieve_type": "root_block",
	},
	task.HTTP: {
		"retrieve_type": "piece",
		"retrieve_size": "1048576",
	},
}

func addErrorResults(
	requester string,
	ipInfo resolver.IPInfo,
	results []interface{},
	document model.DealState,
	providerInfo resolver.MinerInfo,
	location resolver.IPInfo,
	errorCode task.ErrorCode,
	errorMessage string,
) []interface{} {
	for module, metadata := range moduleMetadataMap {
		newMetadata := make(map[string]string)
		for k, v := range metadata {
			newMetadata[k] = v
		}
		newMetadata["deal_id"] = strconv.Itoa(int(document.DealID))
		newMetadata["client"] = document.Client
		results = append(results, task.Result{
			Task: task.Task{
				Requester: requester,
				Module:    module,
				Metadata:  newMetadata,
				Provider: task.Provider{
					ID:         document.Provider,
					PeerID:     providerInfo.PeerId,
					Multiaddrs: convert.MultiaddrsBytesToStringArraySkippingError(providerInfo.Multiaddrs),
					City:       location.City,
					Region:     location.Region,
					Country:    location.Country,
					Continent:  location.Continent,
				},
				Content: task.Content{
					CID: document.Label,
				},
				CreatedAt: time.Now().UTC(),
				Timeout:   env.GetDuration(env.FilplusIntegrationTaskTimeout, 15*time.Second)},
			Retriever: task.Retriever{
				PublicIP:  ipInfo.IP,
				City:      ipInfo.City,
				Region:    ipInfo.Region,
				Country:   ipInfo.Country,
				Continent: ipInfo.Continent,
				ASN:       ipInfo.ASN,
				ISP:       ipInfo.ISP,
				Latitude:  ipInfo.Latitude,
				Longitude: ipInfo.Longitude,
			},
			Result: task.RetrievalResult{
				Success:      false,
				ErrorCode:    errorCode,
				ErrorMessage: errorMessage,
				TTFB:         0,
				Speed:        0,
				Duration:     0,
				Downloaded:   0,
			},
			CreatedAt: time.Now().UTC(),
		})
	}
	return results
}

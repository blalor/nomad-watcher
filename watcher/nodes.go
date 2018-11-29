package watcher

import (
    "time"
    "github.com/sirupsen/logrus"

    "github.com/hashicorp/nomad/api"
)

type NodeEvent struct {
    Timestamp  time.Time `json:"@timestamp"`
    WaitIndex  uint64    `json:"wait_index"`
    Node       *api.Node `json:"node"`
}

func WatchNodes(nodeClient *api.Nodes) <- chan NodeEvent {
    log := logrus.WithFields(logrus.Fields{
        "package": "watcher",
        "fn": "WatchNodes",
    })

    c := make(chan NodeEvent)
    keepWatching := true

    go func() {
        queryOpts := &api.QueryOptions{
            WaitTime: 1 * time.Minute,
            AllowStale: true,
        }

        for keepWatching {
            log.Debugf("retrieving from index %d", queryOpts.WaitIndex)
            nodeStubs, queryMeta, err := nodeClient.List(queryOpts)

            if err != nil {
                log.Errorf("unable to list nodes: %v", err)
                continue
            }

            if queryOpts.WaitIndex > 0 {
                // only emit events after the first run; we're looking for
                // changes

                // the time when the result was retrieved
                now := time.Now()

                // @todo track deleted nodes
                for _, nodeStub := range nodeStubs {
                    if (queryOpts.WaitIndex < nodeStub.CreateIndex) || (queryOpts.WaitIndex < nodeStub.ModifyIndex) {
                        // node was created or updated

                        // retrieve node details
                        node, _, err := nodeClient.Info(nodeStub.ID,  &api.QueryOptions{AllowStale: true})
                        if err != nil {
                            log.Errorf("unable to retrieve node %s: %v", nodeStub.ID, err)
                            continue
                        }

                        c <- NodeEvent{now, queryMeta.LastIndex, node}
                    }
                }
            }

            queryOpts.WaitIndex = queryMeta.LastIndex
        }
    }()

    return c
}

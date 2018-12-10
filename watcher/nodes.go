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

type NodeState struct {
    // copied from the Node
    NodeID string
    NodeName string

    NodeEvent *api.NodeEvent
}

type NodeStateEvent struct {
    Timestamp  time.Time `json:"@timestamp"`
    WaitIndex  uint64    `json:"wait_index"`
    NodeState  NodeState `json:"node_state"`
}

func WatchNodes(nodeClient *api.Nodes) (<- chan NodeEvent, <- chan NodeStateEvent) {
    log := logrus.WithFields(logrus.Fields{
        "package": "watcher",
        "fn": "WatchNodes",
    })

    nodeEventChan := make(chan NodeEvent)
    nodeStateEventChan := make(chan NodeStateEvent)
    keepWatching := true

    go func() {
        // keeping track of the most recently-seen NodeEvent timestamp for each
        // node.  also allows for tracking nodes that are removed.
        nodeEventTimes := make(map[string]time.Time)

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

                // map of nodes ids in the current result
                extantNodes := make(map[string]bool)

                // the time when the result was retrieved
                now := time.Now()

                for _, nodeStub := range nodeStubs {
                    log.Debugf("%+v", nodeStub)

                    nodeId := nodeStub.ID
                    extantNodes[nodeId] = true

                    nodeUpdated := (queryOpts.WaitIndex < nodeStub.CreateIndex) || (queryOpts.WaitIndex < nodeStub.ModifyIndex)

                    if nodeUpdated {
                        // retrieve node details
                        node, _, err := nodeClient.Info(nodeId,  &api.QueryOptions{AllowStale: true})
                        if err != nil {
                            log.Errorf("unable to retrieve node %s: %v", nodeId, err)
                            continue
                        }

                        nodeEventChan <- NodeEvent{now, queryMeta.LastIndex, node}

                        // NodeListStub doesn't contain NodeEvents:
                        // https://github.com/hashicorp/nomad/issues/4976
                        lastNodeEventTime := nodeEventTimes[nodeId]

                        // assumption is node.Events is always ordered
                        for _, event := range node.Events {
                            if event.Timestamp.After(lastNodeEventTime) && event.CreateIndex >= queryMeta.LastIndex {
                                // new NodeStateEvent

                                // emit only if the node was updated
                                if nodeUpdated {
                                    nodeStateEventChan <- NodeStateEvent{
                                        Timestamp: event.Timestamp,
                                        WaitIndex: queryMeta.LastIndex,

                                        NodeState: NodeState{
                                            NodeID:    nodeId,
                                            NodeName:  node.Name,
                                            NodeEvent: event,
                                        },
                                    }
                                }

                                // store the timestamp of the most recent task
                                // event
                                nodeEventTimes[nodeId] = event.Timestamp
                            }
                        }
                    }
                }

                // prune Nodes
                for nodeId, _ := range nodeEventTimes {
                    if _, ok := extantNodes[nodeId]; ! ok {
                        log.Infof("node %s has been deleted", nodeId)

                        delete(nodeEventTimes, nodeId)
                    }
                }

            }

            queryOpts.WaitIndex = queryMeta.LastIndex
        }
    }()

    return nodeEventChan, nodeStateEventChan
}

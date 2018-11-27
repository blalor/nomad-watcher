package watcher

import (
    "time"
    "github.com/sirupsen/logrus"

    "github.com/hashicorp/nomad/api"
)

type DeployEvent struct {
    Timestamp  time.Time       `json:"@timestamp"`
    WaitIndex  uint64          `json:"wait_index"`
    Deployment *api.Deployment `json:"deployment"`
}

func WatchDeployments(deployClient *api.Deployments) <- chan DeployEvent {
    log := logrus.WithFields(logrus.Fields{
        "package": "watcher",
        "fn": "WatchDeployments",
    })

    c := make(chan DeployEvent)
    keepWatching := true

    go func() {
        queryOpts := &api.QueryOptions{
            WaitTime: 1 * time.Minute,
        }

        for keepWatching {
            log.Debugf("retrieving from index %d", queryOpts.WaitIndex)
            deployments, queryMeta, err := deployClient.List(queryOpts)

            if err != nil {
                log.Errorf("unable to list deployments: %v", err)
                continue
            }

            if queryOpts.WaitIndex > 0 {
                // only emit events after the first run; we're looking for
                // changes

                // the time when the result was retrieved
                now := time.Now()

                // @todo track deleted deployments
                for _, deploy := range deployments {
                    if (queryOpts.WaitIndex < deploy.CreateIndex) || (queryOpts.WaitIndex < deploy.ModifyIndex) {
                        // deployment was created or updated
                        c <- DeployEvent{now, queryMeta.LastIndex, deploy}
                    }
                }
            }

            queryOpts.WaitIndex = queryMeta.LastIndex
        }
    }()

    return c
}

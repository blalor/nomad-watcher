package watcher

import (
    "time"
    "github.com/sirupsen/logrus"

    "github.com/hashicorp/nomad/api"
)

type JobEvent struct {
    Timestamp   time.Time `json:"@timestamp"`
    WaitIndex   uint64    `json:"wait_index"`
    Job         *api.Job  `json:"job"`
}

func WatchJobs(jobClient *api.Jobs) <- chan JobEvent {
    log := logrus.WithFields(logrus.Fields{
        "package": "watcher",
        "fn": "WatchJobs",
    })

    c := make(chan JobEvent)
    keepWatching := true

    go func() {
        queryOpts := &api.QueryOptions{
            WaitTime: 1 * time.Minute,
            AllowStale: true,
        }

        for keepWatching {
            log.Debugf("retrieving from index %d", queryOpts.WaitIndex)
            jobStubs, queryMeta, err := jobClient.List(queryOpts)

            if err != nil {
                log.Errorf("unable to list jobs: %v", err)
                continue
            }

            if queryOpts.WaitIndex > 0 {
                // only emit events after the first run; we're looking for
                // changes

                // the time when the result was retrieved
                now := time.Now()

                // @todo track deleted jobs
                for _, jobStub := range jobStubs {
                    if (queryOpts.WaitIndex < jobStub.CreateIndex) || (queryOpts.WaitIndex < jobStub.ModifyIndex) {
                        // job was created or updated

                        // retrieve job details
                        job, _, err := jobClient.Info(jobStub.ID,  &api.QueryOptions{AllowStale: true})
                        if err != nil {
                            log.Errorf("unable to retrieve job %s: %v", jobStub.ID, err)
                            continue
                        }

                        if (job.ModifyIndex != nil) && (jobStub.ModifyIndex != *job.ModifyIndex) {
                            log.Warnf("job %s changed while retrieving details", jobStub.ID)
                        }

                        c <- JobEvent{now, queryMeta.LastIndex, job}
                    }
                }
            }

            queryOpts.WaitIndex = queryMeta.LastIndex
        }
    }()

    return c
}

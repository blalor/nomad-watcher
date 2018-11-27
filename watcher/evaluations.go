package watcher

import (
    "time"
    "github.com/sirupsen/logrus"

    "github.com/hashicorp/nomad/api"
)

type EvalEvent struct {
    Timestamp  time.Time       `json:"@timestamp"`
    WaitIndex  uint64          `json:"wait_index"`
    Evaluation *api.Evaluation `json:"eval"`
}

func WatchEvaluations(evalClient *api.Evaluations) <- chan EvalEvent {
    log := logrus.WithFields(logrus.Fields{
        "package": "watcher",
        "fn": "WatchEvaluations",
    })
    
    c := make(chan EvalEvent)
    keepWatching := true
    
    go func() {
        queryOpts := &api.QueryOptions{
            WaitTime: 1 * time.Minute,
        }
        
        for keepWatching {
            log.Debugf("retrieving from index %d", queryOpts.WaitIndex)
            evals, queryMeta, err := evalClient.List(queryOpts)
            
            if err != nil {
                log.Errorf("unable to list evaluations: %v", err)
                continue
            }
            
            if queryOpts.WaitIndex > 0 {
                // only emit events after the first run; we're looking for
                // changes
                
                // the time when the result was retrieved
                now := time.Now()
                
                // @todo track deleted evals
                for _, eval := range evals {
                    if (queryOpts.WaitIndex < eval.CreateIndex) || (queryOpts.WaitIndex < eval.ModifyIndex) {
                        // evaluation was created or updated
                        c <- EvalEvent{now, queryMeta.LastIndex, eval}
                    }
                }
            }
            
            queryOpts.WaitIndex = queryMeta.LastIndex
        }
    }()
    
    return c
}

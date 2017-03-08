package watcher

import (
    "time"
    "github.com/Sirupsen/logrus"

    "github.com/hashicorp/nomad/api"
)

type AllocEvent struct {
    Timestamp  time.Time                       `json:"@timestamp"`
    WaitIndex  uint64                          `json:"wait_index"`
    AllocationListStub *api.AllocationListStub `json:"alloc"`
}

func WatchAllocations(allocClient *api.Allocations) <- chan AllocEvent {
    log := logrus.WithFields(logrus.Fields{
        "package": "watcher",
        "fn": "WatchAllocations",
    })
    
    c := make(chan AllocEvent)
    keepWatching := true
    
    go func() {
        queryOpts := &api.QueryOptions{
            WaitTime: 1 * time.Minute,
        }
        
        for keepWatching {
            log.Debugf("retrieving from index %d", queryOpts.WaitIndex)
            allocStubs, queryMeta, err := allocClient.List(queryOpts)
            
            if err != nil {
                log.Errorf("unable to list allocations: %v", err)
                continue
            }
            
            if queryOpts.WaitIndex > 0 {
                // only emit events after the first run; we're looking for
                // changes
                
                // the time when the result was retrieved
                now := time.Now()
                
                // @todo track deleted allocations
                for _, allocStub := range allocStubs {
                    ts := now
                    
                    if (allocStub.CreateIndex == allocStub.ModifyIndex) {
                        // allocation just created; use CreateTime field
                        ts = time.Unix(0, allocStub.CreateTime)
                    }
                    
                    if (queryOpts.WaitIndex < allocStub.CreateIndex) || (queryOpts.WaitIndex < allocStub.ModifyIndex) {
                        // allocation was created or updated
                        c <- AllocEvent{ts, queryMeta.LastIndex, allocStub}
                    }
                }
            }
            
            queryOpts.WaitIndex = queryMeta.LastIndex
        }
    }()
    
    return c
}

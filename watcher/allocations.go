package watcher

import (
    "time"
    "github.com/sirupsen/logrus"

    "github.com/hashicorp/nomad/api"
)

type AllocEvent struct {
    Timestamp  time.Time                       `json:"@timestamp"`
    WaitIndex  uint64                          `json:"wait_index"`
    AllocationListStub *api.AllocationListStub `json:"alloc"`
}

// this is a derived event from an allocation. Time comes from TaskEvent.Time.
type TaskStateEvent struct {
    Timestamp  time.Time `json:"@timestamp"`
    WaitIndex  uint64    `json:"wait_index"`
    
    // copied from the Allocation
    JobID              string // "elasticsearch-curator/periodic-1488845700"
    AllocID            string // "29cdfa9e-820a-bab8-4eda-45b000397719"
    AllocName          string // "elasticsearch-curator/periodic-1488845700.curator[0]"
    TaskGroup          string // "curator"
    EvalID             string // "8eb39148-7a00-a164-920f-d59143f72b74"
    NodeID             string // "f8fa01e0-351b-7652-d80e-10547fa3bfe8"
    
    // Task comes from the keys of TaskStates
    Task string
    
    // State and Failed come from the TaskState
    State  string
    Failed bool
    
    // the actual Task Event (phew!)
    TaskEvent *api.TaskEvent
}

func WatchAllocations(allocClient *api.Allocations) (<- chan AllocEvent, <- chan TaskStateEvent) {
    log := logrus.WithFields(logrus.Fields{
        "package": "watcher",
        "fn": "WatchAllocations",
    })
    
    allocEventChan := make(chan AllocEvent)
    taskStateEventChan := make(chan TaskStateEvent)
    
    keepWatching := true
    
    go func() {
        // keeping track of the most recently-seen task state event (TaskEvent)
        // timestamp for each allocation.  also allows for tracking allocations
        // that are removed.
        // @todo possibly need to track per-task times?
        allocTaskEventTimes := make(map[string]int64)
        
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
                
                // map of allocation ids in the current result
                extantAllocs := make(map[string]bool)
                
                for _, allocStub := range allocStubs {
                    ts := now
                    
                    allocId := allocStub.ID
                    extantAllocs[allocId] = true
                    
                    if (allocStub.CreateIndex == allocStub.ModifyIndex) {
                        // allocation just created; use CreateTime field
                        ts = time.Unix(0, allocStub.CreateTime)
                    }
                    
                    // true if allocation was created or updated
                    allocationUpdated := (queryOpts.WaitIndex < allocStub.CreateIndex) || (queryOpts.WaitIndex < allocStub.ModifyIndex)
                    
                    if allocationUpdated {
                        allocEventChan <- AllocEvent{
                            ts,
                            queryMeta.LastIndex,
                            allocStub,
                        }
                    }
                    
                    lastTaskEventTime := allocTaskEventTimes[allocId]
                    for taskName, taskState := range allocStub.TaskStates {
                        for _, taskEvent := range taskState.Events {
                            if taskEvent.Time > lastTaskEventTime {
                                // new TaskEvent
                                
                                // emit only if the allocation was updated
                                if allocationUpdated {
                                    taskStateEventChan <- TaskStateEvent{
                                        Timestamp: time.Unix(0, taskEvent.Time),
                                        WaitIndex: queryMeta.LastIndex,
                                        
                                        JobID:     allocStub.JobID,
                                        AllocID:   allocId,
                                        AllocName: allocStub.Name,
                                        TaskGroup: allocStub.TaskGroup,
                                        EvalID:    allocStub.EvalID,
                                        NodeID:    allocStub.NodeID,

                                        Task:   taskName,
                                        State:  taskState.State,
                                        Failed: taskState.Failed,
                                        
                                        TaskEvent: taskEvent,
                                        
                                    }
                                }

                                // store the timestamp of the most recent task
                                // event; assumption is taskState.Events is always ordered 
                                allocTaskEventTimes[allocId] = taskEvent.Time
                            }
                        }
                    }
                }
                
                // prune allocs
                for allocId, _ := range allocTaskEventTimes {
                    if _, ok := extantAllocs[allocId]; ! ok {
                        log.Infof("allocation %s has been deleted", allocId)
                        
                        delete(allocTaskEventTimes, allocId)
                    }
                }
                
            }
            
            queryOpts.WaitIndex = queryMeta.LastIndex
        }
    }()
    
    return allocEventChan, taskStateEventChan
}
